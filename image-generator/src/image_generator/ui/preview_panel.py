from __future__ import annotations

import base64
import shutil
from pathlib import Path

from PySide6.QtCore import Qt, QUrl
from PySide6.QtGui import QDesktopServices, QPixmap
from PySide6.QtWidgets import (
    QApplication,
    QFileDialog,
    QHBoxLayout,
    QLabel,
    QPushButton,
    QScrollArea,
    QVBoxLayout,
    QWidget,
)

from image_generator.models import GeneratedImage, GenerationTask


class PreviewPanel(QWidget):
    def __init__(self, parent: QWidget | None = None) -> None:
        super().__init__(parent)
        self.current_task: GenerationTask | None = None
        self.current_result: GeneratedImage | None = None
        self._current_pixmap: QPixmap | None = None

        self.image_label = QLabel("No image selected")
        self.image_label.setAlignment(Qt.AlignmentFlag.AlignCenter)
        self.image_label.setMinimumSize(240, 180)
        self.image_label.setScaledContents(False)

        scroll_area = QScrollArea()
        scroll_area.setWidgetResizable(True)
        scroll_area.setWidget(self.image_label)

        self.message_label = QLabel("No image selected")
        self.message_label.setWordWrap(True)
        self.path_label = self.message_label

        self.save_as_button = QPushButton("Save as…")
        self.copy_image_button = QPushButton("Copy image")
        self.open_directory_button = QPushButton("Open directory")
        self.copy_base64_button = QPushButton("Copy base64")
        self.export_base64_button = QPushButton("Export base64…")

        self.save_as_button.clicked.connect(self.save_as)
        self.copy_image_button.clicked.connect(self.copy_image)
        self.open_directory_button.clicked.connect(self.open_directory)
        self.copy_base64_button.clicked.connect(self.copy_base64)
        self.export_base64_button.clicked.connect(self.export_base64)

        action_layout = QHBoxLayout()
        action_layout.addWidget(self.save_as_button)
        action_layout.addWidget(self.copy_image_button)
        action_layout.addWidget(self.open_directory_button)
        action_layout.addWidget(self.copy_base64_button)
        action_layout.addWidget(self.export_base64_button)
        action_layout.addStretch(1)

        layout = QVBoxLayout(self)
        layout.addWidget(scroll_area, 1)
        layout.addWidget(self.message_label)
        layout.addLayout(action_layout)
        self._update_action_state()

    def show_task(self, task: GenerationTask) -> None:
        self.set_task(task)

    def set_task(self, task: GenerationTask | None) -> None:
        self.current_task = task
        self.current_result = task.results[0] if task and task.results else None
        self._current_pixmap = None

        if self.current_result is None:
            self.image_label.clear()
            self.image_label.setText("No image available")
            self.message_label.setText("No image results are available for this task.")
            self._update_action_state()
            return

        pixmap = self._load_pixmap(self.current_result)
        if pixmap is None or pixmap.isNull():
            self.image_label.clear()
            self.image_label.setText("Preview unavailable")
            self.message_label.setText("Preview unavailable for the first result.")
        else:
            self._current_pixmap = pixmap
            self.image_label.setPixmap(pixmap)
            self.message_label.setText(str(self.current_result.path))
        self._update_action_state()

    def save_as(self) -> None:
        result = self.current_result
        if result is None:
            self.message_label.setText("No image available to save.")
            return
        if not result.b64_json and not (result.path and result.path.exists()):
            self.message_label.setText("No image data available to save.")
            return

        suggested_name = result.path.name if result.path else "image.png"
        target, _selected_filter = QFileDialog.getSaveFileName(
            self,
            "Save image as",
            suggested_name,
        )
        if not target:
            return

        target_path = Path(target)
        try:
            if result.path and result.path.exists():
                shutil.copyfile(result.path, target_path)
                self.message_label.setText(f"Saved image to {target_path}")
                return
            if result.b64_json:
                target_path.write_bytes(base64.b64decode(result.b64_json))
                self.message_label.setText(f"Saved image to {target_path}")
                return
        except Exception as exc:
            self.message_label.setText(f"Could not save image: {exc}")
            return

        self.message_label.setText("No image data available to save.")

    def copy_image(self) -> None:
        pixmap = self._current_pixmap
        if pixmap is None or pixmap.isNull():
            self.message_label.setText("No image available to copy.")
            return
        QApplication.clipboard().setPixmap(pixmap)
        self.message_label.setText("Image copied to clipboard.")

    def open_directory(self) -> None:
        result = self.current_result
        if result is None or not result.path:
            self.message_label.setText("No image path available to open.")
            return
        directory = result.path.parent
        if not directory.exists():
            self.message_label.setText("Image directory does not exist.")
            return
        QDesktopServices.openUrl(QUrl.fromLocalFile(str(directory)))

    def copy_base64(self) -> None:
        result = self.current_result
        if result is None or not result.b64_json:
            self.message_label.setText("No base64 data available to copy.")
            return
        QApplication.clipboard().setText(result.b64_json)
        self.message_label.setText("Base64 data copied to clipboard.")

    def export_base64(self) -> None:
        result = self.current_result
        if result is None or not result.b64_json:
            self.message_label.setText("No base64 data available to export.")
            return

        suggested_name = f"{result.path.stem}.b64.txt" if result.path else "image.b64.txt"
        target, _selected_filter = QFileDialog.getSaveFileName(
            self,
            "Export base64 as",
            suggested_name,
            "Text files (*.txt);;All files (*)",
        )
        if not target:
            return

        try:
            Path(target).write_text(result.b64_json, encoding="utf-8")
        except Exception as exc:
            self.message_label.setText(f"Could not export base64 data: {exc}")
            return
        self.message_label.setText(f"Exported base64 data to {target}")

    def resizeEvent(self, event) -> None:  # noqa: N802, ANN001
        super().resizeEvent(event)
        self._fit_pixmap()

    def _load_pixmap(self, result: GeneratedImage) -> QPixmap | None:
        pixmap = QPixmap()
        if result.path and result.path.exists() and pixmap.load(str(result.path)):
            return pixmap
        if result.b64_json:
            try:
                data = base64.b64decode(result.b64_json)
            except Exception:
                return None
            if pixmap.loadFromData(data):
                return pixmap
        return None

    def _fit_pixmap(self) -> None:
        pixmap = self._current_pixmap
        if pixmap is None or pixmap.isNull():
            return
        available = self.image_label.size()
        if available.width() <= 0 or available.height() <= 0:
            self.image_label.setPixmap(pixmap)
            return
        self.image_label.setPixmap(
            pixmap.scaled(
                available,
                Qt.AspectRatioMode.KeepAspectRatio,
                Qt.TransformationMode.SmoothTransformation,
            )
        )

    def _update_action_state(self) -> None:
        has_result = self.current_result is not None
        has_pixmap = self._current_pixmap is not None and not self._current_pixmap.isNull()
        has_path = has_result and bool(
            self.current_result.path and self.current_result.path.exists()
        )
        has_base64 = has_result and bool(self.current_result.b64_json)

        self.save_as_button.setEnabled(has_result and (has_path or has_base64))
        self.copy_image_button.setEnabled(has_pixmap)
        self.open_directory_button.setEnabled(has_path)
        self.copy_base64_button.setEnabled(has_base64)
        self.export_base64_button.setEnabled(has_base64)
