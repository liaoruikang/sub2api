from __future__ import annotations

from pathlib import Path

from PySide6.QtCore import QObject, Signal
from PySide6.QtWidgets import QMessageBox

from image_generator.config import AppConfig, load_config, save_config, validate_config
from image_generator.openai_client import OpenAIImageClient
from image_generator.storage import ImageStorage
from image_generator.task_queue import GenerationQueue
from image_generator.ui.main_window import MainWindow
from image_generator.ui.settings_dialog import SettingsDialog


class QueueBridge(QObject):
    task_updated = Signal(object)


class ImageGeneratorApplication:
    def __init__(self) -> None:
        self.config = load_config()
        self.bridge = QueueBridge()
        self.window: MainWindow | None = None
        self.queue: GenerationQueue | None = None

    def create_window(self) -> MainWindow:
        self.config = self._ensure_valid_config(self.config)
        self.window = MainWindow(self.config)
        self.window.settings_saved.connect(self._save_and_rebuild_services)
        self.window.generate_requested.connect(self.submit_task)
        self.window.batch_requested.connect(self.submit_batch)
        self.bridge.task_updated.connect(self.window.upsert_task)
        self._rebuild_services()
        return self.window

    def submit_task(self, params) -> None:
        if self.queue is None:
            return
        if not params.prompt:
            QMessageBox.information(
                self.window,
                "Missing prompt",
                "Enter a prompt before generating.",
            )
            return
        task = self.queue.submit(params)
        if self.window is not None:
            self.window.upsert_task(task)

    def submit_batch(self, payload) -> None:
        if self.queue is None:
            return
        params, prompts = payload
        tasks = self.queue.submit_batch(prompts, params)
        if self.window is not None:
            for task in tasks:
                self.window.upsert_task(task)

    def _save_and_rebuild_services(self, config: AppConfig) -> None:
        self.config = config.normalized()
        save_config(self.config)
        self._rebuild_services()

    def _rebuild_services(self) -> None:
        client = OpenAIImageClient(self.config)
        storage = ImageStorage(Path(self.config.default_save_dir))
        self.queue = GenerationQueue(
            client,
            storage,
            max_concurrency=self.config.max_concurrency,
            on_task_update=self.bridge.task_updated.emit,
        )

    def _ensure_valid_config(self, config: AppConfig) -> AppConfig:
        current = config.normalized()
        while validate_config(current):
            dialog = SettingsDialog(current)
            if not dialog.exec():
                return current
            current = dialog.to_config()
            messages = validate_config(current)
            if messages:
                QMessageBox.warning(dialog, "Invalid settings", "\n".join(messages))
                continue
            save_config(current)
            return current
        return current
