import pytest
from PySide6.QtWidgets import QDialog

from image_generator import app as app_module
from image_generator.app import ConfigurationCancelled, ImageGeneratorApplication
from image_generator.config import CONFIG_DIR_ENV, AppConfig


def test_first_run_settings_cancel_stops_startup(monkeypatch, tmp_path) -> None:
    monkeypatch.setenv(CONFIG_DIR_ENV, str(tmp_path))

    class CancelledSettingsDialog:
        def __init__(self, config: AppConfig, parent=None) -> None:
            self.config = config
            self.parent = parent

        def exec(self) -> QDialog.DialogCode:
            return QDialog.DialogCode.Rejected

    monkeypatch.setattr(app_module, "SettingsDialog", CancelledSettingsDialog)
    image_app = ImageGeneratorApplication()

    with pytest.raises(ConfigurationCancelled):
        image_app.create_window()

    assert image_app.window is None
    assert image_app.queue is None
