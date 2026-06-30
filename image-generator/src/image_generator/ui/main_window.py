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


class MainWindow(QMainWindow):
    generate_requested = Signal(object)
    batch_requested = Signal(object)
    settings_saved = Signal(object)

    def __init__(self, config: AppConfig | None = None, parent: QWidget | None = None) -> None:
        super().__init__(parent)
        self.setWindowTitle("OpenAI Image Generator")
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
        dialog = SettingsDialog(self.config, self)
        if dialog.exec() != QDialog.DialogCode.Accepted:
            return
        updated = dialog.to_config().normalized()
        messages = validate_config(updated)
        if messages:
            QMessageBox.warning(dialog, "Invalid settings", "\n".join(messages))
            return
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

        splitter = QSplitter(Qt.Orientation.Horizontal)
        splitter.addWidget(self.generation_panel)
        splitter.addWidget(right_splitter)
        splitter.setStretchFactor(0, 1)
        splitter.setStretchFactor(1, 2)

        central = QWidget()
        layout = QHBoxLayout(central)
        layout.addWidget(splitter)
        self.setCentralWidget(central)

    def _build_toolbar(self) -> None:
        toolbar = QToolBar("Main")
        toolbar.setMovable(False)
        self.addToolBar(toolbar)

        self.endpoint_label = QLabel()
        self.api_key_label = QLabel()
        self.concurrency_label = QLabel()
        self.settings_button = QPushButton("Settings")
        self.settings_button.clicked.connect(self.open_settings)

        toolbar.addWidget(self.endpoint_label)
        toolbar.addSeparator()
        toolbar.addWidget(self.api_key_label)
        toolbar.addSeparator()
        toolbar.addWidget(self.concurrency_label)
        toolbar.addSeparator()
        toolbar.addWidget(self.settings_button)

    def _build_status_bar(self) -> None:
        self.status_bar = QStatusBar()
        self.setStatusBar(self.status_bar)
        self.status_bar.showMessage("Ready")

    def _connect_signals(self) -> None:
        self.generation_panel.generate_requested.connect(self._emit_generate_requested)
        self.generation_panel.queue_requested.connect(self._emit_generate_requested)
        self.generation_panel.batch_requested.connect(self._emit_batch_requested)
        self.task_table.selected_task_id.connect(self._show_task_preview)

    def _emit_generate_requested(self, params: GenerationParams) -> None:
        self.generate_requested.emit(params)

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
        endpoint = self.config.base_url or "Not configured"
        self.endpoint_label.setText(f"Endpoint: {endpoint}")
        self.api_key_label.setText(f"API key: {redacted_api_key(self.config.api_key)}")
        self.concurrency_label.setText(f"Concurrency: {self.config.max_concurrency}")
