from __future__ import annotations

from PySide6.QtWidgets import (
    QComboBox,
    QDialog,
    QDialogButtonBox,
    QDoubleSpinBox,
    QFileDialog,
    QFormLayout,
    QHBoxLayout,
    QLabel,
    QLineEdit,
    QPushButton,
    QSpinBox,
    QVBoxLayout,
    QWidget,
)

from image_generator.config import AppConfig
from image_generator.models import EndpointType


class SettingsDialog(QDialog):
    def __init__(self, config: AppConfig, parent: QWidget | None = None) -> None:
        super().__init__(parent)
        self.setWindowTitle("设置")
        self.setMinimumWidth(560)

        normalized = config.normalized()

        title = QLabel("连接设置")
        title.setObjectName("titleLabel")
        subtitle = QLabel("配置 OpenAI 兼容接口与默认保存方式")
        subtitle.setObjectName("subtitleLabel")

        self.base_url_edit = QLineEdit(normalized.base_url)
        self.base_url_edit.setPlaceholderText("例如：https://api.example.com 或 https://api.example.com/v1")

        self.api_key_edit = QLineEdit(normalized.api_key)
        self.api_key_edit.setEchoMode(QLineEdit.EchoMode.Password)
        self.api_key_edit.setPlaceholderText("请输入 API 密钥")

        self.timeout_spin = QDoubleSpinBox()
        self.timeout_spin.setRange(1, 3600)
        self.timeout_spin.setDecimals(1)
        self.timeout_spin.setSingleStep(5)
        self.timeout_spin.setValue(normalized.timeout_seconds)

        self.concurrency_spin = QSpinBox()
        self.concurrency_spin.setRange(1, 64)
        self.concurrency_spin.setValue(normalized.max_concurrency)

        self.save_dir_edit = QLineEdit(str(normalized.default_save_dir))
        self.browse_button = QPushButton("浏览…")
        self.browse_button.setObjectName("secondaryButton")
        self.browse_button.clicked.connect(self._browse_save_dir)

        save_dir_layout = QHBoxLayout()
        save_dir_layout.setContentsMargins(0, 0, 0, 0)
        save_dir_layout.addWidget(self.save_dir_edit, 1)
        save_dir_layout.addWidget(self.browse_button)

        self.endpoint_combo = QComboBox()
        self.endpoint_combo.addItem("图片接口", EndpointType.IMAGES)
        self.endpoint_combo.addItem("聊天补全接口", EndpointType.CHAT_COMPLETIONS)
        endpoint_index = self.endpoint_combo.findData(normalized.default_endpoint_type)
        if endpoint_index >= 0:
            self.endpoint_combo.setCurrentIndex(endpoint_index)

        self.model_edit = QLineEdit(normalized.default_model)

        form = QFormLayout()
        form.setContentsMargins(16, 16, 16, 16)
        form.setSpacing(10)
        form.addRow("接口地址", self.base_url_edit)
        form.addRow("API 密钥", self.api_key_edit)
        form.addRow("超时时间（秒）", self.timeout_spin)
        form.addRow("最大并发数", self.concurrency_spin)
        form.addRow("默认保存目录", save_dir_layout)
        form.addRow("默认接口", self.endpoint_combo)
        form.addRow("默认模型", self.model_edit)

        buttons = QDialogButtonBox(
            QDialogButtonBox.StandardButton.Ok | QDialogButtonBox.StandardButton.Cancel
        )
        buttons.accepted.connect(self.accept)
        buttons.rejected.connect(self.reject)

        layout = QVBoxLayout(self)
        layout.setContentsMargins(18, 18, 18, 18)
        layout.setSpacing(12)
        layout.addWidget(title)
        layout.addWidget(subtitle)
        layout.addSpacing(6)
        layout.addLayout(form)
        layout.addWidget(buttons)

    def to_config(self) -> AppConfig:
        return AppConfig(
            base_url=self.base_url_edit.text(),
            api_key=self.api_key_edit.text(),
            timeout_seconds=self.timeout_spin.value(),
            max_concurrency=self.concurrency_spin.value(),
            default_save_dir=self.save_dir_edit.text(),
            default_endpoint_type=self._current_endpoint_type(),
            default_model=self.model_edit.text(),
        ).normalized()

    def _browse_save_dir(self) -> None:
        selected = QFileDialog.getExistingDirectory(
            self,
            "选择默认保存目录",
            self.save_dir_edit.text(),
        )
        if selected:
            self.save_dir_edit.setText(selected)

    def _current_endpoint_type(self) -> EndpointType:
        value = self.endpoint_combo.currentData()
        try:
            return EndpointType(value)
        except (TypeError, ValueError):
            return EndpointType.IMAGES
