from pathlib import Path

from PySide6.QtCore import Qt
from PySide6.QtWidgets import QComboBox, QFileDialog, QLineEdit

from image_generator.config import AppConfig
from image_generator.models import EndpointType, GenerationParams
from image_generator.ui.generation_panel import GenerationPanel
from image_generator.ui.settings_dialog import SettingsDialog


def set_combo_data(combo: QComboBox, data: object) -> None:
    index = combo.findData(data)
    assert index >= 0
    combo.setCurrentIndex(index)


def test_settings_dialog_round_trips_normalized_config(qtbot, tmp_path: Path) -> None:
    config = AppConfig(
        base_url=" https://api.example.com/v1/ ",
        api_key=" sk-test-secret ",
        timeout_seconds=45.5,
        max_concurrency=5,
        default_save_dir=tmp_path / "images",
        default_endpoint_type=EndpointType.CHAT_COMPLETIONS,
        default_model=" image-model ",
    )
    dialog = SettingsDialog(config)
    qtbot.addWidget(dialog)

    assert dialog.api_key_edit.echoMode() is QLineEdit.EchoMode.Password
    assert dialog.to_config().default_endpoint_type is EndpointType.CHAT_COMPLETIONS

    dialog.base_url_edit.setText(" https://other.example.com/// ")
    dialog.api_key_edit.setText(" sk-updated-secret ")
    dialog.timeout_spin.setValue(90.0)
    dialog.concurrency_spin.setValue(7)
    dialog.save_dir_edit.setText(str(tmp_path / "updated-images"))
    set_combo_data(dialog.endpoint_combo, EndpointType.IMAGES)
    dialog.model_edit.setText(" updated-model ")

    updated = dialog.to_config()

    assert updated == AppConfig(
        base_url="https://other.example.com",
        api_key="sk-updated-secret",
        timeout_seconds=90.0,
        max_concurrency=7,
        default_save_dir=tmp_path / "updated-images",
        default_endpoint_type=EndpointType.IMAGES,
        default_model="updated-model",
    )


def test_settings_dialog_browse_button_updates_save_directory(qtbot, monkeypatch, tmp_path: Path) -> None:
    selected_dir = tmp_path / "selected"
    dialog = SettingsDialog(AppConfig(default_save_dir=tmp_path / "initial"))
    qtbot.addWidget(dialog)

    monkeypatch.setattr(
        QFileDialog,
        "getExistingDirectory",
        lambda *args, **kwargs: str(selected_dir),
    )

    qtbot.mouseClick(dialog.browse_button, Qt.MouseButton.LeftButton)

    assert dialog.save_dir_edit.text() == str(selected_dir)


def test_generation_panel_builds_generation_params(qtbot) -> None:
    panel = GenerationPanel(default_model="gpt-image-1")
    qtbot.addWidget(panel)

    panel.prompt_edit.setPlainText("A watercolor fox")
    panel.model_edit.setText("custom-image-model")
    set_combo_data(panel.endpoint_combo, EndpointType.CHAT_COMPLETIONS)
    panel.size_combo.setCurrentText("1792x1024")
    panel.count_spin.setValue(3)
    panel.quality_combo.setCurrentText("hd")
    panel.style_combo.setCurrentText("vivid")
    panel.response_format_combo.setCurrentText("url")
    panel.stream_check.setChecked(True)

    params = panel.to_params()

    assert params == GenerationParams(
        prompt="A watercolor fox",
        endpoint_type=EndpointType.CHAT_COMPLETIONS,
        model="custom-image-model",
        size="1792x1024",
        n=3,
        quality="hd",
        style="vivid",
        response_format="url",
        stream=True,
    )


def test_generation_panel_buttons_emit_current_params(qtbot) -> None:
    panel = GenerationPanel(default_model="image-model")
    qtbot.addWidget(panel)
    panel.prompt_edit.setPlainText("City skyline")

    with qtbot.waitSignal(panel.generate_requested, timeout=1000) as generate_signal:
        qtbot.mouseClick(panel.generate_button, Qt.MouseButton.LeftButton)
    with qtbot.waitSignal(panel.queue_requested, timeout=1000) as queue_signal:
        qtbot.mouseClick(panel.queue_button, Qt.MouseButton.LeftButton)
    with qtbot.waitSignal(panel.batch_requested, timeout=1000) as batch_signal:
        qtbot.mouseClick(panel.batch_button, Qt.MouseButton.LeftButton)

    generated = generate_signal.args[0]
    queued = queue_signal.args[0]
    batched = batch_signal.args[0]

    assert isinstance(generated, GenerationParams)
    assert generated.prompt == "City skyline"
    assert queued == generated
    assert batched == generated
