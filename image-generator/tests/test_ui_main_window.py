from __future__ import annotations

import base64
from pathlib import Path

from PySide6.QtCore import Qt
from PySide6.QtGui import QDesktopServices
from PySide6.QtWidgets import QApplication, QDialog, QFileDialog

from image_generator.config import AppConfig, redacted_api_key
from image_generator.models import GeneratedImage, GenerationParams, GenerationTask, TaskStatus
from image_generator.ui import main_window as main_window_module
from image_generator.ui.main_window import MainWindow
from image_generator.ui.preview_panel import PreviewPanel
from image_generator.ui.task_table import TaskTable

PNG_B64 = (
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJ"
    "AAAADUlEQVR4nGP4z8DwHwAFAAH/iZk9HQAAAABJRU5ErkJggg=="
)


def make_task(
    task_id: str = "task-1",
    prompt: str = "A watercolor fox in a quiet forest",
    *,
    status: TaskStatus = TaskStatus.QUEUED,
    model: str = "gpt-image-1",
    events: list[str] | None = None,
    error: str | None = None,
    results: list[GeneratedImage] | None = None,
) -> GenerationTask:
    task = GenerationTask(
        id=task_id,
        params=GenerationParams(prompt=prompt, model=model),
        status=status,
        events=list(events or []),
        error=error,
        results=list(results or []),
    )
    return task


def write_png(path: Path) -> bytes:
    data = base64.b64decode(PNG_B64)
    path.write_bytes(data)
    return data


def test_task_table_upserts_selects_and_removes_rows(qtbot, tmp_path: Path) -> None:
    table = TaskTable()
    qtbot.addWidget(table)
    image_path = tmp_path / "first.png"
    write_png(image_path)

    first = make_task(
        "task-1",
        "A very long prompt that should be summarized for the table display",
        status=TaskStatus.GENERATING,
        model="model-a",
        events=["queued", "halfway done"],
        results=[
            GeneratedImage(
                path=image_path,
                source_type="b64_json",
                source_ref="b64",
                b64_json=PNG_B64,
            )
        ],
    )
    second = make_task(
        "task-2",
        "Second prompt",
        status=TaskStatus.FAILED,
        model="model-b",
        events=["queued"],
        error="provider failed",
    )

    table.upsert_task(first)
    table.upsert_task(second)
    table.upsert_task(
        make_task(
            "task-1",
            "Updated prompt",
            status=TaskStatus.COMPLETED,
            model="model-c",
        )
    )

    assert [table.horizontalHeaderItem(column).text() for column in range(table.columnCount())] == [
        "ID",
        "Status",
        "Prompt summary",
        "Model",
        "Result count",
        "Latest event",
        "Error",
    ]
    assert table.rowCount() == 2
    assert table.item(0, 0).text() == "task-1"
    assert table.item(0, 1).text() == "completed"
    assert table.item(0, 2).text() == "Updated prompt"
    assert table.item(0, 3).text() == "model-c"
    assert table.item(0, 4).text() == "0"
    assert table.item(1, 0).text() == "task-2"
    assert table.item(1, 5).text() == "queued"
    assert table.item(1, 6).text() == "provider failed"

    table.remove_task("task-1")

    assert table.rowCount() == 1
    assert table.item(0, 0).text() == "task-2"
    assert table.row_by_task_id == {"task-2": 0}

    table.clearSelection()
    with qtbot.waitSignal(table.selected_task_id, timeout=1000) as selected_signal:
        table.selectRow(0)

    assert selected_signal.args == ["task-2"]


def test_preview_panel_handles_empty_state_and_first_result_actions(
    qtbot,
    monkeypatch,
    tmp_path: Path,
) -> None:
    panel = PreviewPanel()
    qtbot.addWidget(panel)

    panel.set_task(make_task(results=[]))
    panel.save_as()
    panel.copy_image()
    panel.copy_base64()
    panel.open_directory()

    assert "No image" in panel.message_label.text()

    source_path = tmp_path / "source.png"
    image_bytes = write_png(source_path)
    task = make_task(
        results=[
            GeneratedImage(
                path=source_path,
                source_type="b64_json",
                source_ref="b64",
                b64_json=PNG_B64,
            )
        ]
    )

    panel.set_task(task)

    assert panel.current_task is task
    assert panel.image_label.pixmap() is not None
    assert not panel.image_label.pixmap().isNull()

    QApplication.clipboard().clear()
    panel.copy_base64()
    assert QApplication.clipboard().text() == PNG_B64

    copied_path = tmp_path / "copied.png"
    monkeypatch.setattr(
        QFileDialog,
        "getSaveFileName",
        lambda *args, **kwargs: (str(copied_path), ""),
    )
    panel.save_as()
    assert copied_path.read_bytes() == image_bytes

    exported_path = tmp_path / "image.b64.txt"
    monkeypatch.setattr(
        QFileDialog,
        "getSaveFileName",
        lambda *args, **kwargs: (str(exported_path), ""),
    )
    panel.export_base64()
    assert exported_path.read_text(encoding="utf-8") == PNG_B64

    opened_urls = []
    monkeypatch.setattr(
        QDesktopServices,
        "openUrl",
        lambda url: opened_urls.append(url) or True,
    )
    panel.open_directory()
    assert opened_urls
    assert Path(opened_urls[0].toLocalFile()) == source_path.parent


def test_preview_panel_actions_do_not_crash_without_existing_image_data(
    qtbot,
    monkeypatch,
    tmp_path: Path,
) -> None:
    panel = PreviewPanel()
    qtbot.addWidget(panel)
    task = make_task(
        results=[
            GeneratedImage(
                path=tmp_path / "missing.png",
                source_type="url",
                source_ref="https://example.com/missing.png",
            )
        ]
    )

    monkeypatch.setattr(
        QFileDialog,
        "getSaveFileName",
        lambda *args, **kwargs: (_ for _ in ()).throw(
            AssertionError("file dialog should not open without image data")
        ),
    )

    panel.set_task(task)
    panel.save_as()
    panel.copy_image()
    panel.copy_base64()
    panel.export_base64()
    panel.open_directory()

    assert "base64" in panel.message_label.text()



def test_main_window_shows_redacted_status_and_forwards_generation_signals(qtbot) -> None:
    config = AppConfig(
        base_url="https://api.example.com/v1",
        api_key="sk-secret-value-123456",
        max_concurrency=7,
        default_model="image-model",
    )
    window = MainWindow(config=config)
    qtbot.addWidget(window)

    status_text = " ".join(
        [
            window.endpoint_label.text(),
            window.api_key_label.text(),
            window.concurrency_label.text(),
        ]
    )
    assert config.api_key not in status_text
    assert redacted_api_key(config.api_key) in status_text
    assert "https://api.example.com/v1" in status_text
    assert "7" in status_text

    window.generation_panel.prompt_edit.setPlainText("Single prompt")

    with qtbot.waitSignal(window.generate_requested, timeout=1000) as generate_signal:
        qtbot.mouseClick(window.generation_panel.generate_button, Qt.MouseButton.LeftButton)
    with qtbot.waitSignal(window.generate_requested, timeout=1000) as queue_signal:
        qtbot.mouseClick(window.generation_panel.queue_button, Qt.MouseButton.LeftButton)

    generated = generate_signal.args[0]
    queued = queue_signal.args[0]
    assert generated.prompt == "Single prompt"
    assert generated.model == "image-model"
    assert queued == generated

    window.generation_panel.prompt_edit.setPlainText("First prompt\n\n Second prompt \n")
    with qtbot.waitSignal(window.batch_requested, timeout=1000) as batch_signal:
        qtbot.mouseClick(window.generation_panel.batch_button, Qt.MouseButton.LeftButton)

    batch_params, prompts = batch_signal.args[0]
    assert prompts == ["First prompt", "Second prompt"]
    assert batch_params.model == "image-model"
    assert batch_params.prompt == ""


def test_main_window_selection_updates_preview_and_settings_signal(
    qtbot,
    monkeypatch,
    tmp_path: Path,
) -> None:
    window = MainWindow(
        config=AppConfig(api_key="sk-original-secret", default_model="image-model")
    )
    qtbot.addWidget(window)
    image_path = tmp_path / "preview.png"
    write_png(image_path)
    task = make_task(
        results=[
            GeneratedImage(
                path=image_path,
                source_type="b64_json",
                source_ref="b64",
                b64_json=PNG_B64,
            )
        ]
    )

    window.upsert_task(task)

    with qtbot.waitSignal(window.task_table.selected_task_id, timeout=1000):
        window.task_table.selectRow(0)

    assert window.preview_panel.current_task is task
    assert window.preview_panel.image_label.pixmap() is not None
    assert not window.preview_panel.image_label.pixmap().isNull()

    updated_config = AppConfig(
        base_url="https://new.example.com/v1",
        api_key="sk-updated-secret",
        max_concurrency=3,
        default_model="new-model",
    ).normalized()

    class AcceptedSettingsDialog:
        def __init__(self, config: AppConfig, parent=None) -> None:
            self.config = config
            self.parent = parent

        def exec(self) -> QDialog.DialogCode:
            return QDialog.DialogCode.Accepted

        def to_config(self) -> AppConfig:
            return updated_config

    monkeypatch.setattr(main_window_module, "SettingsDialog", AcceptedSettingsDialog)

    with qtbot.waitSignal(window.settings_saved, timeout=1000) as settings_signal:
        qtbot.mouseClick(window.settings_button, Qt.MouseButton.LeftButton)

    assert settings_signal.args == [updated_config]
    assert "sk-updated-secret" not in window.api_key_label.text()
    assert redacted_api_key("sk-updated-secret") in window.api_key_label.text()
    assert window.generation_panel.model_edit.text() == "new-model"
