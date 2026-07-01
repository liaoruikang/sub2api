from __future__ import annotations

from PySide6.QtWidgets import (
    QComboBox,
    QDialog,
    QDialogButtonBox,
    QDoubleSpinBox,
    QFileDialog,
    QFormLayout,
    QHBoxLayout,
    QLineEdit,
    QPushButton,
    QSpinBox,
    QVBoxLayout,
    QWidget,
)

from image_generator.config import AppConfig
from image_generator.models import EndpointType


def _endpoint_type(value: object) -> EndpointType:
    try:
        return EndpointType(value)
    except (TypeError, ValueError):
        return EndpointType.IMAGES


class SettingsDialog(QDialog):
    def __init__(self, config: AppConfig, parent: QWidget | None = None) -> None:
        super().__init__(parent)
        self.setWindowTitle("Settings")

        normalized = config.normalized()

        self.base_url_edit = QLineEdit(normalized.base_url)
        self.api_key_edit = QLineEdit(normalized.api_key)
        self.api_key_edit.setEchoMode(QLineEdit.EchoMode.Password)

        self.timeout_spin = QDoubleSpinBox()
        self.timeout_spin.setRange(1, 3600)
        self.timeout_spin.setDecimals(1)
        self.timeout_spin.setSingleStep(5)
        self.timeout_spin.setValue(normalized.timeout_seconds)

        self.concurrency_spin = QSpinBox()
        self.concurrency_spin.setRange(1, 64)
        self.concurrency_spin.setValue(normalized.max_concurrency)

        self.save_dir_edit = QLineEdit(str(normalized.default_save_dir))
        self.browse_button = QPushButton("Browse…")
        self.browse_button.clicked.connect(self._browse_save_dir)

        save_dir_layout = QHBoxLayout()
        save_dir_layout.addWidget(self.save_dir_edit, 1)
        save_dir_layout.addWidget(self.browse_button)

        self.endpoint_combo = QComboBox()
        self.endpoint_combo.addItem("Images", EndpointType.IMAGES)
        self.endpoint_combo.addItem("Chat completions", EndpointType.CHAT_COMPLETIONS)
        endpoint_index = self.endpoint_combo.findData(normalized.default_endpoint_type)
        if endpoint_index >= 0:
            self.endpoint_combo.setCurrentIndex(endpoint_index)

        self.model_edit = QLineEdit(normalized.default_model)

        form = QFormLayout()
        form.addRow("Base URL", self.base_url_edit)
        form.addRow("API key", self.api_key_edit)
        form.addRow("Timeout seconds", self.timeout_spin)
        form.addRow("Max concurrency", self.concurrency_spin)
        form.addRow("Default save directory", save_dir_layout)
        form.addRow("Default endpoint", self.endpoint_combo)
        form.addRow("Default model", self.model_edit)

        buttons = QDialogButtonBox(
            QDialogButtonBox.StandardButton.Ok | QDialogButtonBox.StandardButton.Cancel
        )
        buttons.accepted.connect(self.accept)
        buttons.rejected.connect(self.reject)

        layout = QVBoxLayout(self)
        layout.addLayout(form)
        layout.addWidget(buttons)

    def to_config(self) -> AppConfig:
        endpoint_type = _endpoint_type(self.endpoint_combo.currentData())

        return AppConfig(
            base_url=self.base_url_edit.text(),
            api_key=self.api_key_edit.text(),
            timeout_seconds=self.timeout_spin.value(),
            max_concurrency=self.concurrency_spin.value(),
            default_save_dir=self.save_dir_edit.text(),
            default_endpoint_type=endpoint_type,
            default_model=self.model_edit.text(),
        ).normalized()

    def _browse_save_dir(self) -> None:
        selected = QFileDialog.getExistingDirectory(
            self,
            "Choose default save directory",
            self.save_dir_edit.text(),
        )
        if selected:
            self.save_dir_edit.setText(selected)
