from __future__ import annotations

import base64
from pathlib import Path

from PySide6.QtCore import Qt
from PySide6.QtGui import QDesktopServices, QPixmap
from PySide6.QtWidgets import QApplication, QDialog, QFileDialog, QMessageBox

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
    return GenerationTask(
        id=task_id,
        params=GenerationParams(prompt=prompt, model=model),
        status=status,
        events=list(events or []),
        error=error,
        results=list(results or []),
    )


def write_png(path: Path) -> bytes:
    data = base64.b64decode(PNG_B64)
    path.write_bytes(data)
    return data


def write_large_png(path: Path) -> bytes:
    pixmap = QPixmap(1200, 900)
    pixmap.fill(Qt.GlobalColor.red)
    assert pixmap.save(str(path), "PNG")
    return path.read_bytes()


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
        "任务 ID",
        "状态",
        "提示词摘要",
        "模型",
        "结果数量",
        "最新事件",
        "错误",
    ]
    assert table.rowCount() == 2
    assert table.item(0, 0).text() == "task-1"
    assert table.item(0, 1).text() == "已完成"
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

    assert "没有" in panel.message_label.text()

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


def test_preview_panel_zoom_controls(qtbot, tmp_path: Path) -> None:
    panel = PreviewPanel()
    qtbot.addWidget(panel)
    panel.resize(420, 320)
    image_path = tmp_path / "large.png"
    write_large_png(image_path)

    panel.set_task(
        make_task(
            results=[
                GeneratedImage(
                    path=image_path,
                    source_type="url",
                    source_ref="https://example.com/large.png",
                )
            ]
        )
    )

    fitted = panel.image_label.pixmap()
    assert fitted is not None
    assert not fitted.isNull()
    fitted_width = fitted.width()
    assert panel.zoom_label.text() == "适应窗口"

    panel.zoom_in()
    zoomed = panel.image_label.pixmap()
    assert zoomed is not None
    assert zoomed.width() > fitted_width
    assert panel.zoom_label.text() == "125%"

    panel.actual_size()
    assert panel.zoom_label.text() == "100%"

    panel.fit_to_window()
    refitted = panel.image_label.pixmap()
    assert refitted is not None
    assert refitted.width() <= zoomed.width()
    assert panel.zoom_label.text() == "适应窗口"


def test_preview_panel_can_select_each_generated_result(qtbot, tmp_path: Path) -> None:
    panel = PreviewPanel()
    qtbot.addWidget(panel)
    first_path = tmp_path / "first.png"
    second_path = tmp_path / "second.png"
    first_bytes = write_png(first_path)
    second_bytes = write_large_png(second_path)
    first_b64 = base64.b64encode(first_bytes).decode("ascii")
    second_b64 = base64.b64encode(second_bytes).decode("ascii")

    panel.set_task(
        make_task(
            results=[
                GeneratedImage(
                    path=first_path,
                    source_type="b64_json",
                    source_ref="first",
                    b64_json=first_b64,
                ),
                GeneratedImage(
                    path=second_path,
                    source_type="b64_json",
                    source_ref="second",
                    b64_json=second_b64,
                ),
            ]
        )
    )

    assert panel.result_combo.count() == 2
    assert panel.result_combo.isEnabled()
    assert panel.result_combo.itemText(0).startswith("结果 1")
    assert panel.current_result is not None
    assert panel.current_result.path == first_path

    panel.result_combo.setCurrentIndex(1)

    assert panel.current_result is not None
    assert panel.current_result.path == second_path
    assert str(second_path) in panel.path_label.text()
    QApplication.clipboard().clear()
    panel.copy_base64()
    assert QApplication.clipboard().text() == second_b64


def test_preview_panel_url_result_can_copy_and_export_base64(
    qtbot,
    monkeypatch,
    tmp_path: Path,
) -> None:
    panel = PreviewPanel()
    qtbot.addWidget(panel)
    image_path = tmp_path / "downloaded.png"
    image_bytes = write_png(image_path)
    expected_b64 = base64.b64encode(image_bytes).decode("ascii")
    panel.set_task(
        make_task(
            results=[
                GeneratedImage(
                    path=image_path,
                    source_type="url",
                    source_ref="https://example.com/downloaded.png",
                )
            ]
        )
    )

    assert panel.copy_base64_button.isEnabled()
    assert panel.export_base64_button.isEnabled()

    QApplication.clipboard().clear()
    panel.copy_base64()
    assert QApplication.clipboard().text() == expected_b64

    exported_path = tmp_path / "downloaded.b64.txt"
    monkeypatch.setattr(
        QFileDialog,
        "getSaveFileName",
        lambda *args, **kwargs: (str(exported_path), ""),
    )
    panel.export_base64()
    assert exported_path.read_text(encoding="utf-8") == expected_b64


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

    assert "Base64" in panel.message_label.text()


def test_main_window_shows_redacted_status_and_forwards_generation_signals(qtbot) -> None:
    config = AppConfig(
        base_url="https://api.example.com/v1",
        api_key="sk-secret-value-123456",
        max_concurrency=7,
        default_model="image-model",
    )
    window = MainWindow(config=config)
    qtbot.addWidget(window)

    assert window.windowTitle() == "图片生成器"
    assert window.settings_button.objectName() == "settingsButton"
    assert window.generation_panel.objectName() == "sidebarCard"
    assert window.preview_panel.scroll_area.objectName() == "previewCanvas"
    status_text = " ".join(
        [
            window.endpoint_label.text(),
            window.api_key_label.text(),
            window.concurrency_label.text(),
        ]
    )
    assert config.api_key not in status_text
    assert redacted_api_key(config.api_key) in status_text
    assert "接口地址：https://api.example.com/v1" in status_text
    assert "并发数：7" in status_text

    window.generation_panel.prompt_edit.setPlainText("Single prompt")

    with qtbot.waitSignal(window.generate_requested, timeout=1000) as generate_signal:
        qtbot.mouseClick(window.generation_panel.generate_button, Qt.MouseButton.LeftButton)
    with qtbot.waitSignal(window.queue_requested, timeout=1000) as queue_signal:
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


def test_main_window_rejects_invalid_settings_without_saving(
    qtbot,
    monkeypatch,
) -> None:
    original_config = AppConfig(
        base_url="https://api.example.com",
        api_key="sk-original-secret",
        default_model="image-model",
    ).normalized()
    window = MainWindow(config=original_config)
    qtbot.addWidget(window)
    invalid_config = AppConfig(base_url="", api_key="", default_model="bad-model")
    seen_configs: list[AppConfig] = []

    class InvalidThenCancelledSettingsDialog:
        def __init__(self, config: AppConfig, parent=None) -> None:
            self.config = config
            self.parent = parent
            seen_configs.append(config)

        def exec(self) -> QDialog.DialogCode:
            if len(seen_configs) == 1:
                return QDialog.DialogCode.Accepted
            return QDialog.DialogCode.Rejected

        def to_config(self) -> AppConfig:
            return invalid_config

    warnings: list[tuple[str, str]] = []
    monkeypatch.setattr(main_window_module, "SettingsDialog", InvalidThenCancelledSettingsDialog)
    monkeypatch.setattr(
        QMessageBox,
        "warning",
        lambda _parent, title, message: warnings.append((title, message)),
    )

    qtbot.mouseClick(window.settings_button, Qt.MouseButton.LeftButton)

    assert window.config == original_config
    assert window.generation_panel.model_edit.text() == "image-model"
    assert seen_configs == [original_config, invalid_config.normalized()]
    assert warnings == [("设置无效", "接口地址不能为空\nAPI 密钥不能为空")]


def test_main_window_does_not_replace_selected_preview_when_other_task_completes(
    qtbot,
    tmp_path: Path,
) -> None:
    window = MainWindow(config=AppConfig())
    qtbot.addWidget(window)
    first_path = tmp_path / "first.png"
    second_path = tmp_path / "second.png"
    write_png(first_path)
    write_png(second_path)

    selected_task = make_task("task-a", "selected")
    completing_task = make_task("task-b", "background")
    window.upsert_task(selected_task)
    window.upsert_task(completing_task)

    with qtbot.waitSignal(window.task_table.selected_task_id, timeout=1000):
        window.task_table.selectRow(0)

    assert window.preview_panel.current_task is selected_task

    completed_background_task = make_task(
        "task-b",
        "background",
        status=TaskStatus.COMPLETED,
        results=[
            GeneratedImage(
                path=second_path,
                source_type="b64_json",
                source_ref="b64",
                b64_json=PNG_B64,
            )
        ],
    )
    window.upsert_task(completed_background_task)

    assert window.task_table.selected_task_id is not None
    assert window.preview_panel.current_task is selected_task

    completed_selected_task = make_task(
        "task-a",
        "selected",
        status=TaskStatus.COMPLETED,
        results=[
            GeneratedImage(
                path=first_path,
                source_type="b64_json",
                source_ref="b64",
                b64_json=PNG_B64,
            )
        ],
    )
    window.upsert_task(completed_selected_task)

    assert window.preview_panel.current_task is completed_selected_task
