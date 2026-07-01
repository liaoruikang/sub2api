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


class ConfigurationCancelled(RuntimeError):
    pass


class QueueBridge(QObject):
    task_updated = Signal(object)


class ImageGeneratorApplication:
    def __init__(self) -> None:
        self.config = load_config().normalized()
        self.bridge = QueueBridge()
        self.window: MainWindow | None = None
        self.queue: GenerationQueue | None = None

    def create_window(self) -> MainWindow:
        self.window = MainWindow(self.config)
        self.window.settings_saved.connect(self._save_and_rebuild_services)
        self.window.generate_requested.connect(self.submit_task)
        self.window.queue_requested.connect(self.submit_task)
        self.window.batch_requested.connect(self.submit_batch)
        self.bridge.task_updated.connect(self.window.upsert_task)
        self._rebuild_services()
        return self.window

    def submit_task(self, params) -> None:
        if self.queue is None:
            return
        if not self._ensure_ready_to_generate():
            return
        if not params.prompt:
            QMessageBox.information(
                self.window,
                "缺少提示词",
                "请先输入提示词再开始生成。",
            )
            return
        task = self.queue.submit(params)
        if self.window is not None:
            self.window.upsert_task(task)

    def submit_batch(self, payload) -> None:
        if self.queue is None:
            return
        if not self._ensure_ready_to_generate():
            return
        params, prompts = payload
        if not prompts:
            QMessageBox.information(
                self.window,
                "缺少提示词",
                "请在提示词输入框中按行输入批量提示词。",
            )
            return
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

    def _ensure_ready_to_generate(self) -> bool:
        messages = _localized_config_messages(validate_config(self.config))
        if not messages:
            return True
        QMessageBox.warning(
            self.window,
            "设置不完整",
            "请先在“设置”中补全以下内容：\n" + "\n".join(messages),
        )
        return False

    def _ensure_valid_config(self, config: AppConfig) -> AppConfig:
        current = config.normalized()
        while validate_config(current):
            dialog = SettingsDialog(current)
            if not dialog.exec():
                raise ConfigurationCancelled("Initial settings were cancelled")
            current = dialog.to_config()
            messages = validate_config(current)
            if messages:
                QMessageBox.warning(
                    dialog,
                    "设置无效",
                    "\n".join(_localized_config_messages(messages)),
                )
                continue
            save_config(current)
            return current
        return current


def _localized_config_messages(messages: list[str]) -> list[str]:
    translations = {
        "Base URL is required": "接口地址不能为空",
        "API key is required": "API 密钥不能为空",
        "Timeout must be at least 1 second": "超时时间至少为 1 秒",
        "Max concurrency must be at least 1": "最大并发数至少为 1",
        "Default model is required": "默认模型不能为空",
        "Default save directory is required": "默认保存目录不能为空",
    }
    return [translations.get(message, message) for message in messages]
