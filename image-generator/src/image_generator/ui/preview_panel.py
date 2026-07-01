from __future__ import annotations

import base64
import shutil
from pathlib import Path

from PySide6.QtCore import Qt, QUrl
from PySide6.QtGui import QDesktopServices, QPixmap
from PySide6.QtWidgets import (
    QApplication,
    QComboBox,
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
        self._zoom_factor = 1.0
        self._fit_to_window = True

        self.image_label = QLabel("未选择图片")
        self.image_label.setAlignment(Qt.AlignmentFlag.AlignCenter)
        self.image_label.setMinimumSize(360, 260)
        self.image_label.setScaledContents(False)

        self.scroll_area = QScrollArea()
        self.scroll_area.setWidgetResizable(True)
        self.scroll_area.setAlignment(Qt.AlignmentFlag.AlignCenter)
        self.scroll_area.setWidget(self.image_label)

        self.result_combo = QComboBox()
        self.result_combo.currentIndexChanged.connect(self._select_result)

        self.zoom_out_button = QPushButton("缩小")
        self.zoom_in_button = QPushButton("放大")
        self.actual_size_button = QPushButton("100%")
        self.fit_button = QPushButton("适应窗口")
        self.zoom_label = QLabel("适应窗口")
        self.zoom_out_button.clicked.connect(self.zoom_out)
        self.zoom_in_button.clicked.connect(self.zoom_in)
        self.actual_size_button.clicked.connect(self.actual_size)
        self.fit_button.clicked.connect(self.fit_to_window)

        zoom_layout = QHBoxLayout()
        zoom_layout.setContentsMargins(0, 0, 0, 0)
        zoom_layout.setSpacing(8)
        zoom_layout.addWidget(QLabel("预览结果"))
        zoom_layout.addWidget(self.result_combo, 1)
        zoom_layout.addWidget(self.zoom_out_button)
        zoom_layout.addWidget(self.zoom_in_button)
        zoom_layout.addWidget(self.actual_size_button)
        zoom_layout.addWidget(self.fit_button)
        zoom_layout.addWidget(self.zoom_label)

        self.path_label = QLabel("未选择图片")
        self.path_label.setWordWrap(True)
        self.message_label = QLabel("未选择图片")
        self.message_label.setWordWrap(True)

        self.save_as_button = QPushButton("另存为…")
        self.copy_image_button = QPushButton("复制图片")
        self.open_directory_button = QPushButton("打开目录")
        self.copy_base64_button = QPushButton("复制 Base64")
        self.export_base64_button = QPushButton("导出 Base64…")

        self.save_as_button.clicked.connect(self.save_as)
        self.copy_image_button.clicked.connect(self.copy_image)
        self.open_directory_button.clicked.connect(self.open_directory)
        self.copy_base64_button.clicked.connect(self.copy_base64)
        self.export_base64_button.clicked.connect(self.export_base64)

        action_layout = QHBoxLayout()
        action_layout.setSpacing(8)
        action_layout.addWidget(self.save_as_button)
        action_layout.addWidget(self.copy_image_button)
        action_layout.addWidget(self.open_directory_button)
        action_layout.addWidget(self.copy_base64_button)
        action_layout.addWidget(self.export_base64_button)
        action_layout.addStretch(1)

        layout = QVBoxLayout(self)
        layout.setContentsMargins(16, 16, 16, 16)
        layout.setSpacing(10)
        layout.addLayout(zoom_layout)
        layout.addWidget(self.scroll_area, 1)
        layout.addWidget(self.path_label)
        layout.addWidget(self.message_label)
        layout.addLayout(action_layout)
        self._populate_results(None)
        self._update_action_state()

    def show_task(self, task: GenerationTask) -> None:
        self.set_task(task)

    def set_task(self, task: GenerationTask | None) -> None:
        self.current_task = task
        self._populate_results(task)
        self._select_result(0 if task and task.results else -1)

    def zoom_in(self) -> None:
        if self._current_pixmap is None:
            return
        self._fit_to_window = False
        self._zoom_factor = min(self._zoom_factor * 1.25, 5.0)
        self._render_pixmap()

    def zoom_out(self) -> None:
        if self._current_pixmap is None:
            return
        self._fit_to_window = False
        self._zoom_factor = max(self._zoom_factor / 1.25, 0.1)
        self._render_pixmap()

    def actual_size(self) -> None:
        if self._current_pixmap is None:
            return
        self._fit_to_window = False
        self._zoom_factor = 1.0
        self._render_pixmap()

    def fit_to_window(self) -> None:
        if self._current_pixmap is None:
            return
        self._fit_to_window = True
        self._render_pixmap()

    def save_as(self) -> None:
        result = self.current_result
        if result is None:
            self.message_label.setText("没有可保存的图片。")
            return
        if not result.b64_json and not (result.path and result.path.exists()):
            self.message_label.setText("没有可保存的图片数据。")
            return

        suggested_name = result.path.name if result.path else "image.png"
        target, _selected_filter = QFileDialog.getSaveFileName(
            self,
            "图片另存为",
            suggested_name,
        )
        if not target:
            return

        target_path = Path(target)
        try:
            if result.path and result.path.exists():
                shutil.copyfile(result.path, target_path)
                self.message_label.setText(f"已保存图片到 {target_path}")
                return
            if result.b64_json:
                target_path.write_bytes(base64.b64decode(result.b64_json))
                self.message_label.setText(f"已保存图片到 {target_path}")
                return
        except Exception as exc:
            self.message_label.setText(f"保存图片失败：{exc}")
            return

        self.message_label.setText("没有可保存的图片数据。")

    def copy_image(self) -> None:
        pixmap = self._current_pixmap
        if pixmap is None or pixmap.isNull():
            self.message_label.setText("没有可复制的图片。")
            return
        QApplication.clipboard().setPixmap(pixmap)
        self.message_label.setText("图片已复制到剪贴板。")

    def open_directory(self) -> None:
        result = self.current_result
        if result is None or not result.path:
            self.message_label.setText("没有可打开的图片路径。")
            return
        directory = result.path.parent
        if not directory.exists():
            self.message_label.setText("图片所在目录不存在。")
            return
        QDesktopServices.openUrl(QUrl.fromLocalFile(str(directory)))

    def copy_base64(self) -> None:
        b64_data = self._current_base64()
        if not b64_data:
            self.message_label.setText("没有可复制的 Base64 数据。")
            return
        QApplication.clipboard().setText(b64_data)
        self.message_label.setText("Base64 数据已复制到剪贴板。")

    def export_base64(self) -> None:
        result = self.current_result
        b64_data = self._current_base64()
        if result is None or not b64_data:
            self.message_label.setText("没有可导出的 Base64 数据。")
            return

        suggested_name = f"{result.path.stem}.b64.txt" if result.path else "image.b64.txt"
        target, _selected_filter = QFileDialog.getSaveFileName(
            self,
            "导出 Base64",
            suggested_name,
            "文本文件 (*.txt);;所有文件 (*)",
        )
        if not target:
            return

        try:
            Path(target).write_text(b64_data, encoding="utf-8")
        except Exception as exc:
            self.message_label.setText(f"导出 Base64 失败：{exc}")
            return
        self.message_label.setText(f"已导出 Base64 到 {target}")

    def resizeEvent(self, event) -> None:  # noqa: N802, ANN001
        super().resizeEvent(event)
        if self._fit_to_window:
            self._render_pixmap()

    def _populate_results(self, task: GenerationTask | None) -> None:
        self.result_combo.blockSignals(True)
        self.result_combo.clear()
        if task is not None:
            for index, result in enumerate(task.results):
                label = f"结果 {index + 1}"
                if result.path:
                    label = f"{label}: {result.path.name}"
                self.result_combo.addItem(label, index)
        self.result_combo.setEnabled(self.result_combo.count() > 1)
        self.result_combo.blockSignals(False)

    def _select_result(self, index: int) -> None:
        results = self.current_task.results if self.current_task is not None else []
        self._current_pixmap = None
        self._zoom_factor = 1.0
        self._fit_to_window = True
        if index < 0 or index >= len(results):
            self.current_result = None
            self.image_label.clear()
            self.image_label.setText("没有可预览的图片")
            self.path_label.setText("当前任务没有图片结果")
            self.message_label.setText("当前任务没有可用的图片结果。")
            self._update_action_state()
            self._update_zoom_label()
            return

        self.current_result = results[index]
        pixmap = self._load_pixmap(self.current_result)
        if pixmap is None or pixmap.isNull():
            self.image_label.clear()
            self.image_label.setText("无法预览")
            self.path_label.setText(str(self.current_result.path))
            self.message_label.setText(f"无法预览第 {index + 1} 个结果。")
        else:
            self._current_pixmap = pixmap
            self._render_pixmap()
            self.path_label.setText(str(self.current_result.path))
            self.message_label.setText(f"正在显示第 {index + 1} / {len(results)} 个结果。")
        self._update_action_state()

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

    def _current_base64(self) -> str | None:
        result = self.current_result
        if result is None:
            return None
        if result.b64_json:
            return result.b64_json
        if result.path and result.path.exists():
            try:
                return base64.b64encode(result.path.read_bytes()).decode("ascii")
            except Exception:
                return None
        return None

    def _render_pixmap(self) -> None:
        pixmap = self._current_pixmap
        if pixmap is None or pixmap.isNull():
            self._update_zoom_label()
            return
        if self._fit_to_window:
            available = self.scroll_area.viewport().size()
            if available.width() <= 0 or available.height() <= 0:
                scaled = pixmap
            else:
                scaled = pixmap.scaled(
                    available,
                    Qt.AspectRatioMode.KeepAspectRatio,
                    Qt.TransformationMode.SmoothTransformation,
                )
            self.image_label.setPixmap(scaled)
            self.image_label.resize(scaled.size())
        else:
            width = max(1, int(pixmap.width() * self._zoom_factor))
            height = max(1, int(pixmap.height() * self._zoom_factor))
            scaled = pixmap.scaled(
                width,
                height,
                Qt.AspectRatioMode.KeepAspectRatio,
                Qt.TransformationMode.SmoothTransformation,
            )
            self.image_label.setPixmap(scaled)
            self.image_label.resize(scaled.size())
        self._update_zoom_label()

    def _update_action_state(self) -> None:
        has_result = self.current_result is not None
        has_pixmap = self._current_pixmap is not None and not self._current_pixmap.isNull()
        has_path = has_result and bool(
            self.current_result.path and self.current_result.path.exists()
        )
        has_base64 = bool(self._current_base64())

        self.save_as_button.setEnabled(has_result and (has_path or has_base64))
        self.copy_image_button.setEnabled(has_pixmap)
        self.open_directory_button.setEnabled(has_path)
        self.copy_base64_button.setEnabled(has_base64)
        self.export_base64_button.setEnabled(has_base64)
        self.zoom_in_button.setEnabled(has_pixmap)
        self.zoom_out_button.setEnabled(has_pixmap)
        self.actual_size_button.setEnabled(has_pixmap)
        self.fit_button.setEnabled(has_pixmap)

    def _update_zoom_label(self) -> None:
        if self._current_pixmap is None:
            self.zoom_label.setText("无图片")
        elif self._fit_to_window:
            self.zoom_label.setText("适应窗口")
        else:
            self.zoom_label.setText(f"{self._zoom_factor:.0%}")
