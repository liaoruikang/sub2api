from __future__ import annotations

from dataclasses import replace

from PySide6.QtCore import Qt, Signal
from PySide6.QtWidgets import (
    QDialog,
    QHBoxLayout,
    QLabel,
    QMainWindow,
    QMessageBox,
    QPushButton,
    QSizePolicy,
    QSplitter,
    QStatusBar,
    QToolBar,
    QWidget,
)

from image_generator.config import AppConfig, redacted_api_key, validate_config
from image_generator.models import GenerationParams, GenerationTask
from image_generator.ui.generation_panel import GenerationPanel
from image_generator.ui.preview_panel import PreviewPanel
from image_generator.ui.settings_dialog import SettingsDialog
from image_generator.ui.task_table import TaskTable

CONFIG_MESSAGE_LABELS = {
    "Base URL is required": "接口地址不能为空",
    "API key is required": "API 密钥不能为空",
    "Timeout must be at least 1 second": "超时时间至少为 1 秒",
    "Max concurrency must be at least 1": "最大并发数至少为 1",
    "Default model is required": "默认模型不能为空",
    "Default save directory is required": "默认保存目录不能为空",
}


class MainWindow(QMainWindow):
    generate_requested = Signal(object)
    queue_requested = Signal(object)
    batch_requested = Signal(object)
    settings_saved = Signal(object)

    def __init__(self, config: AppConfig | None = None, parent: QWidget | None = None) -> None:
        super().__init__(parent)
        self.setWindowTitle("图片生成器")
        self.setMinimumSize(1100, 720)
        self.resize(1280, 800)
        self.config = (config or AppConfig()).normalized()
        self.tasks: dict[str, GenerationTask] = {}

        self.generation_panel = GenerationPanel(default_model=self.config.default_model)
        self.task_table = TaskTable()
        self.preview_panel = PreviewPanel()

        self._apply_config_to_generation_panel()
        self._build_layout()
        self._build_toolbar()
        self._build_status_bar()
        self._connect_signals()
        self._update_status_labels()

    def upsert_task(self, task: GenerationTask) -> None:
        self.tasks[task.id] = task
        self.task_table.upsert_task(task)
        if self.preview_panel.current_task is None and task.results:
            self.preview_panel.set_task(task)
        elif self.preview_panel.current_task and self.preview_panel.current_task.id == task.id:
            self.preview_panel.set_task(task)

    def remove_task(self, task_id: str) -> None:
        self.tasks.pop(task_id, None)
        self.task_table.remove_task(task_id)
        if self.preview_panel.current_task and self.preview_panel.current_task.id == task_id:
            self.preview_panel.set_task(None)

    def open_settings(self) -> None:
        current = self.config
        while True:
            dialog = SettingsDialog(current, self)
            if dialog.exec() != QDialog.DialogCode.Accepted:
                return
            updated = dialog.to_config().normalized()
            messages = validate_config(updated)
            if not messages:
                break
            QMessageBox.warning(dialog, "设置无效", "\n".join(_localized_messages(messages)))
            current = updated
        self.config = updated
        self._apply_config_to_generation_panel()
        self._update_status_labels()
        self.settings_saved.emit(self.config)

    def _build_layout(self) -> None:
        right_splitter = QSplitter(Qt.Orientation.Vertical)
        right_splitter.addWidget(self.task_table)
        right_splitter.addWidget(self.preview_panel)
        right_splitter.setStretchFactor(0, 2)
        right_splitter.setStretchFactor(1, 3)
        right_splitter.setSizes([280, 520])

        splitter = QSplitter(Qt.Orientation.Horizontal)
        splitter.addWidget(self.generation_panel)
        splitter.addWidget(right_splitter)
        splitter.setStretchFactor(0, 0)
        splitter.setStretchFactor(1, 1)
        splitter.setSizes([380, 900])

        central = QWidget()
        layout = QHBoxLayout(central)
        layout.setContentsMargins(12, 12, 12, 12)
        layout.setSpacing(10)
        layout.addWidget(splitter)
        self.setCentralWidget(central)

    def _build_toolbar(self) -> None:
        toolbar = QToolBar("主工具栏")
        toolbar.setMovable(False)
        self.addToolBar(toolbar)

        self.endpoint_label = QLabel()
        self.api_key_label = QLabel()
        self.concurrency_label = QLabel()
        self.settings_button = QPushButton("设置")
        self.settings_button.clicked.connect(self.open_settings)

        spacer = QWidget()
        spacer.setSizePolicy(QSizePolicy.Policy.Expanding, QSizePolicy.Policy.Preferred)

        toolbar.addWidget(self.endpoint_label)
        toolbar.addSeparator()
        toolbar.addWidget(self.api_key_label)
        toolbar.addSeparator()
        toolbar.addWidget(self.concurrency_label)
        toolbar.addWidget(spacer)
        toolbar.addWidget(self.settings_button)

    def _build_status_bar(self) -> None:
        self.status_bar = QStatusBar()
        self.setStatusBar(self.status_bar)
        self.status_bar.showMessage("就绪")

    def _connect_signals(self) -> None:
        self.generation_panel.generate_requested.connect(self._emit_generate_requested)
        self.generation_panel.queue_requested.connect(self._emit_queue_requested)
        self.generation_panel.batch_requested.connect(self._emit_batch_requested)
        self.task_table.selected_task_id.connect(self._show_task_preview)

    def _emit_generate_requested(self, params: GenerationParams) -> None:
        self.generate_requested.emit(params)

    def _emit_queue_requested(self, params: GenerationParams) -> None:
        self.queue_requested.emit(params)

    def _emit_batch_requested(self, params: GenerationParams) -> None:
        prompts = [line.strip() for line in params.prompt.splitlines() if line.strip()]
        self.batch_requested.emit((replace(params, prompt=""), prompts))

    def _show_task_preview(self, task_id: str) -> None:
        self.preview_panel.set_task(self.tasks.get(task_id))

    def _apply_config_to_generation_panel(self) -> None:
        self.generation_panel.model_edit.setText(self.config.default_model)
        endpoint_index = self.generation_panel.endpoint_combo.findData(
            self.config.default_endpoint_type
        )
        if endpoint_index >= 0:
            self.generation_panel.endpoint_combo.setCurrentIndex(endpoint_index)

    def _update_status_labels(self) -> None:
        endpoint = self.config.base_url or "未配置"
        self.endpoint_label.setText(f"接口地址：{endpoint}")
        self.api_key_label.setText(f"API 密钥：{redacted_api_key(self.config.api_key)}")
        self.concurrency_label.setText(f"并发数：{self.config.max_concurrency}")


def _localized_messages(messages: list[str]) -> list[str]:
    return [CONFIG_MESSAGE_LABELS.get(message, message) for message in messages]
