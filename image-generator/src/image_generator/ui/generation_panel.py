from __future__ import annotations

from PySide6.QtCore import Qt, Signal
from PySide6.QtWidgets import (
    QCheckBox,
    QComboBox,
    QFormLayout,
    QGroupBox,
    QHBoxLayout,
    QLabel,
    QLineEdit,
    QPushButton,
    QSpinBox,
    QTextEdit,
    QVBoxLayout,
    QWidget,
)

from image_generator.models import EndpointType, GenerationParams

SIZE_PRESETS = [
    "512x512",
    "768x768",
    "1024x1024",
    "1024x1536",
    "1536x1024",
    "1024x1792",
    "1792x1024",
    "1536x1536",
]


class GenerationPanel(QWidget):
    generate_requested = Signal(object)
    queue_requested = Signal(object)
    batch_requested = Signal(object)

    def __init__(self, default_model: str = "gpt-image-1", parent: QWidget | None = None) -> None:
        super().__init__(parent)
        self.setObjectName("sidebarCard")

        title = QLabel("创作")
        title.setObjectName("titleLabel")
        subtitle = QLabel("描述画面，选择模型与输出参数")
        subtitle.setObjectName("subtitleLabel")

        self.prompt_edit = QTextEdit()
        self.prompt_edit.setObjectName("promptEditor")
        self.prompt_edit.setPlaceholderText("请输入图片描述，例如：一只水彩风格的狐狸")
        self.prompt_edit.setMinimumHeight(150)

        self.model_edit = QLineEdit(default_model.strip() or "gpt-image-1")

        self.endpoint_combo = QComboBox()
        self.endpoint_combo.addItem("图片接口", EndpointType.IMAGES)
        self.endpoint_combo.addItem("聊天补全接口", EndpointType.CHAT_COMPLETIONS)

        self.size_combo = QComboBox()
        self.size_combo.setEditable(True)
        self.size_combo.addItems(SIZE_PRESETS)
        self.size_combo.setCurrentText("1024x1024")

        self.count_spin = QSpinBox()
        self.count_spin.setRange(1, 10)
        self.count_spin.setValue(1)

        self.quality_combo = QComboBox()
        self.quality_combo.addItem("标准", "standard")
        self.quality_combo.addItem("高清", "hd")
        self.quality_combo.setEditable(True)

        self.style_combo = QComboBox()
        self.style_combo.addItem("自然", "natural")
        self.style_combo.addItem("鲜明", "vivid")
        self.style_combo.setEditable(True)

        self.response_format_combo = QComboBox()
        self.response_format_combo.addItem("Base64", "b64_json")
        self.response_format_combo.addItem("图片 URL", "url")
        self.response_format_combo.setEditable(True)

        self.stream_check = QCheckBox("流式响应")

        prompt_group = QGroupBox("提示词")
        prompt_group.setObjectName("card")
        prompt_layout = QVBoxLayout(prompt_group)
        prompt_layout.setContentsMargins(14, 16, 14, 14)
        prompt_layout.addWidget(self.prompt_edit)

        form = QFormLayout()
        form.setContentsMargins(14, 16, 14, 14)
        form.setSpacing(12)
        form.setLabelAlignment(Qt.AlignmentFlag.AlignLeft)
        form.addRow("模型", self.model_edit)
        form.addRow("接口类型", self.endpoint_combo)
        form.addRow("图片尺寸", self.size_combo)
        form.addRow("生成数量", self.count_spin)
        form.addRow("质量", self.quality_combo)
        form.addRow("风格", self.style_combo)
        form.addRow("响应格式", self.response_format_combo)
        form.addRow("流式", self.stream_check)
        params_group = QGroupBox("生成参数")
        params_group.setObjectName("card")
        params_group.setLayout(form)

        self.generate_button = QPushButton("立即生成")
        self.queue_button = QPushButton("加入队列")
        self.batch_button = QPushButton("批量生成")
        self.generate_button.setObjectName("primaryButton")
        self.queue_button.setObjectName("secondaryButton")
        self.batch_button.setObjectName("ghostButton")
        self.generate_button.clicked.connect(self._emit_generate_requested)
        self.queue_button.clicked.connect(self._emit_queue_requested)
        self.batch_button.clicked.connect(self._emit_batch_requested)

        button_layout = QHBoxLayout()
        button_layout.setContentsMargins(14, 14, 14, 14)
        button_layout.setSpacing(10)
        button_layout.addWidget(self.generate_button, 2)
        button_layout.addWidget(self.queue_button, 1)
        button_layout.addWidget(self.batch_button, 1)
        actions_group = QGroupBox("操作")
        actions_group.setObjectName("card")
        actions_group.setLayout(button_layout)

        layout = QVBoxLayout(self)
        layout.setContentsMargins(20, 20, 20, 20)
        layout.setSpacing(14)
        layout.addWidget(title)
        layout.addWidget(subtitle)
        layout.addSpacing(6)
        layout.addWidget(prompt_group)
        layout.addWidget(params_group)
        layout.addWidget(actions_group)
        layout.addStretch(1)

    def to_params(self) -> GenerationParams:
        return GenerationParams(
            prompt=self.prompt_edit.toPlainText().strip(),
            endpoint_type=self._current_endpoint_type(),
            model=self.model_edit.text().strip() or "gpt-image-1",
            size=self.size_combo.currentText().strip() or "1024x1024",
            n=self.count_spin.value(),
            quality=self._combo_value(self.quality_combo, "standard"),
            style=self._combo_value(self.style_combo, "natural"),
            response_format=self._combo_value(self.response_format_combo, "b64_json"),
            stream=self.stream_check.isChecked(),
        )

    def _emit_generate_requested(self) -> None:
        self.generate_requested.emit(self.to_params())

    def _emit_queue_requested(self) -> None:
        self.queue_requested.emit(self.to_params())

    def _emit_batch_requested(self) -> None:
        self.batch_requested.emit(self.to_params())

    def _current_endpoint_type(self) -> EndpointType:
        value = self.endpoint_combo.currentData()
        try:
            return EndpointType(value)
        except (TypeError, ValueError):
            return EndpointType.IMAGES

    @staticmethod
    def _combo_value(combo: QComboBox, fallback: str) -> str:
        data = combo.currentData()
        if isinstance(data, str) and data:
            return data
        return combo.currentText().strip() or fallback
