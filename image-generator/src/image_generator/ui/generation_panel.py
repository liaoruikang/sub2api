from __future__ import annotations

from PySide6.QtCore import Signal
from PySide6.QtWidgets import (
    QCheckBox,
    QComboBox,
    QFormLayout,
    QHBoxLayout,
    QLineEdit,
    QPushButton,
    QSpinBox,
    QTextEdit,
    QVBoxLayout,
    QWidget,
)

from image_generator.models import EndpointType, GenerationParams


def _endpoint_type(value: object) -> EndpointType:
    try:
        return EndpointType(value)
    except (TypeError, ValueError):
        return EndpointType.IMAGES


class GenerationPanel(QWidget):
    generate_requested = Signal(object)
    queue_requested = Signal(object)
    batch_requested = Signal(object)

    def __init__(self, default_model: str = "gpt-image-1", parent: QWidget | None = None) -> None:
        super().__init__(parent)

        self.prompt_edit = QTextEdit()
        self.prompt_edit.setPlaceholderText("Describe the image to generate")

        self.model_edit = QLineEdit(default_model.strip() or "gpt-image-1")

        self.endpoint_combo = QComboBox()
        self.endpoint_combo.addItem("Images", EndpointType.IMAGES)
        self.endpoint_combo.addItem("Chat completions", EndpointType.CHAT_COMPLETIONS)

        self.size_combo = QComboBox()
        self.size_combo.setEditable(True)
        self.size_combo.addItems(["1024x1024", "1024x1792", "1792x1024"])

        self.count_spin = QSpinBox()
        self.count_spin.setRange(1, 10)
        self.count_spin.setValue(1)

        self.quality_combo = QComboBox()
        self.quality_combo.setEditable(True)
        self.quality_combo.addItems(["standard", "hd"])

        self.style_combo = QComboBox()
        self.style_combo.setEditable(True)
        self.style_combo.addItems(["natural", "vivid"])

        self.response_format_combo = QComboBox()
        self.response_format_combo.setEditable(True)
        self.response_format_combo.addItems(["b64_json", "url"])

        self.stream_check = QCheckBox("Stream response")

        form = QFormLayout()
        form.addRow("Prompt", self.prompt_edit)
        form.addRow("Model", self.model_edit)
        form.addRow("Endpoint", self.endpoint_combo)
        form.addRow("Size", self.size_combo)
        form.addRow("Count", self.count_spin)
        form.addRow("Quality", self.quality_combo)
        form.addRow("Style", self.style_combo)
        form.addRow("Response format", self.response_format_combo)
        form.addRow("Streaming", self.stream_check)

        self.generate_button = QPushButton("Generate")
        self.queue_button = QPushButton("Queue")
        self.batch_button = QPushButton("Batch")
        self.generate_button.clicked.connect(self._emit_generate_requested)
        self.queue_button.clicked.connect(self._emit_queue_requested)
        self.batch_button.clicked.connect(self._emit_batch_requested)

        button_layout = QHBoxLayout()
        button_layout.addStretch(1)
        button_layout.addWidget(self.generate_button)
        button_layout.addWidget(self.queue_button)
        button_layout.addWidget(self.batch_button)

        layout = QVBoxLayout(self)
        layout.addLayout(form)
        layout.addLayout(button_layout)

    def to_params(self) -> GenerationParams:
        endpoint_type = _endpoint_type(self.endpoint_combo.currentData())

        return GenerationParams(
            prompt=self.prompt_edit.toPlainText().strip(),
            endpoint_type=endpoint_type,
            model=self.model_edit.text().strip() or "gpt-image-1",
            size=self.size_combo.currentText().strip() or "1024x1024",
            n=self.count_spin.value(),
            quality=self.quality_combo.currentText().strip() or "standard",
            style=self.style_combo.currentText().strip() or "natural",
            response_format=self.response_format_combo.currentText().strip() or "b64_json",
            stream=self.stream_check.isChecked(),
        )

    def _emit_generate_requested(self) -> None:
        self.generate_requested.emit(self.to_params())

    def _emit_queue_requested(self) -> None:
        self.queue_requested.emit(self.to_params())

    def _emit_batch_requested(self) -> None:
        self.batch_requested.emit(self.to_params())
