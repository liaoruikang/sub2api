from __future__ import annotations

from PySide6.QtCore import Qt, Signal
from PySide6.QtGui import QColor
from PySide6.QtWidgets import (
    QAbstractItemView,
    QHeaderView,
    QTableWidget,
    QTableWidgetItem,
    QWidget,
)

from image_generator.models import GenerationTask, TaskStatus

STATUS_LABELS = {
    TaskStatus.QUEUED: "排队中",
    TaskStatus.CONNECTING: "连接中",
    TaskStatus.GENERATING: "生成中",
    TaskStatus.SAVING: "保存中",
    TaskStatus.COMPLETED: "已完成",
    TaskStatus.FAILED: "失败",
    TaskStatus.CANCELLED: "已取消",
}

STATUS_COLORS = {
    TaskStatus.GENERATING: "#9b7a3c",
    TaskStatus.SAVING: "#9b7a3c",
    TaskStatus.COMPLETED: "#39765a",
    TaskStatus.FAILED: "#b05246",
    TaskStatus.CANCELLED: "#70685d",
}


class TaskTable(QTableWidget):
    selected_task_id = Signal(str)

    HEADERS = [
        "任务 ID",
        "状态",
        "提示词摘要",
        "模型",
        "结果数量",
        "最新事件",
        "错误",
    ]

    def __init__(self, parent: QWidget | None = None) -> None:
        super().__init__(0, len(self.HEADERS), parent)
        self.setObjectName("workspaceCard")
        self.row_by_task_id: dict[str, int] = {}

        self.setAlternatingRowColors(True)
        self.setShowGrid(False)
        self.verticalHeader().setDefaultSectionSize(42)
        self.setHorizontalHeaderLabels(self.HEADERS)
        self.setSelectionBehavior(QAbstractItemView.SelectionBehavior.SelectRows)
        self.setSelectionMode(QAbstractItemView.SelectionMode.SingleSelection)
        self.setEditTriggers(QAbstractItemView.EditTrigger.NoEditTriggers)
        self.verticalHeader().setVisible(False)
        self.horizontalHeader().setSectionResizeMode(QHeaderView.ResizeMode.ResizeToContents)
        self.horizontalHeader().setSectionResizeMode(2, QHeaderView.ResizeMode.Stretch)
        self.horizontalHeader().setSectionResizeMode(5, QHeaderView.ResizeMode.Stretch)
        self.horizontalHeader().setSectionResizeMode(6, QHeaderView.ResizeMode.Stretch)
        self.currentCellChanged.connect(self._emit_selected_task_id)

    def upsert_task(self, task: GenerationTask) -> None:
        row = self.row_by_task_id.get(task.id)
        if row is None:
            row = self.rowCount()
            self.insertRow(row)
            self.row_by_task_id[task.id] = row

        values = [
            task.id,
            self._status_text(task.status),
            self._prompt_summary(task.params.prompt),
            task.params.model,
            str(len(task.results)),
            task.events[-1] if task.events else "",
            task.error or "",
        ]
        for column, value in enumerate(values):
            item = self._item(value)
            if column == 1 and isinstance(task.status, TaskStatus):
                color = STATUS_COLORS.get(task.status)
                if color:
                    item.setForeground(QColor(color))
            self.setItem(row, column, item)

    def remove_task(self, task_id: str) -> None:
        row = self.row_by_task_id.get(task_id)
        if row is None:
            return
        self.removeRow(row)
        self._rebuild_row_mapping()

    def task_id_at_row(self, row: int) -> str | None:
        if row < 0 or row >= self.rowCount():
            return None
        item = self.item(row, 0)
        if item is None:
            return None
        return item.text()

    def _emit_selected_task_id(
        self,
        row: int,
        _column: int,
        _old_row: int,
        _old_column: int,
    ) -> None:
        task_id = self.task_id_at_row(row)
        if task_id:
            self.selected_task_id.emit(task_id)

    def _rebuild_row_mapping(self) -> None:
        self.row_by_task_id = {}
        for row in range(self.rowCount()):
            task_id = self.task_id_at_row(row)
            if task_id:
                self.row_by_task_id[task_id] = row

    @staticmethod
    def _item(value: str) -> QTableWidgetItem:
        item = QTableWidgetItem(value)
        item.setFlags(Qt.ItemFlag.ItemIsSelectable | Qt.ItemFlag.ItemIsEnabled)
        return item

    @staticmethod
    def _status_text(status: TaskStatus | str) -> str:
        if isinstance(status, TaskStatus):
            return STATUS_LABELS.get(status, status.value)
        return str(status)

    @staticmethod
    def _prompt_summary(prompt: str) -> str:
        single_line = " ".join(prompt.split())
        if len(single_line) <= 80:
            return single_line
        return f"{single_line[:77]}..."
