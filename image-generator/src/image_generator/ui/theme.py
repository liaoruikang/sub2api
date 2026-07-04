from __future__ import annotations

from PySide6.QtWidgets import QApplication

BACKGROUND = "#f4f0e8"
SURFACE = "#fbf8f1"
SURFACE_ELEVATED = "#fffdf8"
SURFACE_SUNKEN = "#ece6dc"
CANVAS = "#151719"
BORDER = "#ddd3c4"
BORDER_STRONG = "#c9bca9"
TEXT = "#1f211f"
TEXT_MUTED = "#70685d"
ACCENT = "#9b7a3c"
ACCENT_DARK = "#725824"
SUCCESS = "#39765a"
DANGER = "#b05246"
SELECTION = "#efe3cb"


def apply_app_theme(app: QApplication) -> None:
    app.setApplicationName("图片生成器")
    app.setStyle("Fusion")
    app.setStyleSheet(_stylesheet())


def _stylesheet() -> str:
    return f"""
    QMainWindow, QDialog {{
        background: {BACKGROUND};
        color: {TEXT};
        font-family: "Microsoft YaHei UI", "Microsoft YaHei", "Segoe UI";
        font-size: 13px;
    }}

    QWidget {{
        color: {TEXT};
        selection-background-color: {SELECTION};
        selection-color: {TEXT};
    }}

    QLabel#titleLabel {{
        color: {TEXT};
        font-size: 22px;
        font-weight: 700;
        letter-spacing: 1px;
    }}

    QLabel#subtitleLabel, QLabel#mutedLabel, QLabel#configStatusLabel,
    QLabel#metadataLabel, QLabel#messageLabel {{
        color: {TEXT_MUTED};
    }}

    QToolBar#mainToolbar {{
        background: {BACKGROUND};
        border: none;
        padding: 12px 18px 4px 18px;
        spacing: 12px;
    }}

    QStatusBar {{
        background: {BACKGROUND};
        color: {TEXT_MUTED};
        border-top: 1px solid {BORDER};
    }}

    QGroupBox, QWidget#card, QWidget#sidebarCard, QWidget#workspaceCard {{
        background: {SURFACE_ELEVATED};
        border: 1px solid {BORDER};
        border-radius: 14px;
    }}

    QGroupBox {{
        margin-top: 18px;
        padding: 16px 12px 12px 12px;
        font-weight: 600;
    }}

    QGroupBox::title {{
        subcontrol-origin: margin;
        left: 14px;
        padding: 0 6px;
        color: {TEXT_MUTED};
        background: {SURFACE_ELEVATED};
    }}

    QLineEdit, QTextEdit, QComboBox, QSpinBox, QDoubleSpinBox {{
        background: {SURFACE};
        color: {TEXT};
        border: 1px solid {BORDER};
        border-radius: 10px;
        padding: 8px 10px;
        min-height: 22px;
    }}

    QTextEdit {{
        padding: 12px;
        line-height: 150%;
    }}

    QLineEdit:focus, QTextEdit:focus, QComboBox:focus,
    QSpinBox:focus, QDoubleSpinBox:focus {{
        border: 1px solid {ACCENT};
        background: {SURFACE_ELEVATED};
    }}

    QComboBox::drop-down {{
        border: none;
        width: 28px;
    }}

    QComboBox QAbstractItemView {{
        background: {SURFACE_ELEVATED};
        color: {TEXT};
        border: 1px solid {BORDER};
        selection-background-color: {SELECTION};
        outline: none;
    }}

    QPushButton {{
        background: {SURFACE};
        color: {TEXT};
        border: 1px solid {BORDER_STRONG};
        border-radius: 10px;
        padding: 8px 14px;
        min-height: 24px;
        font-weight: 600;
    }}

    QPushButton:hover {{
        background: {SURFACE_ELEVATED};
        border-color: {ACCENT};
    }}

    QPushButton:pressed {{
        background: {SURFACE_SUNKEN};
    }}

    QPushButton:disabled {{
        color: #aaa195;
        background: #eee8dd;
        border-color: #e2d9cb;
    }}

    QPushButton#primaryButton {{
        background: {TEXT};
        color: {SURFACE_ELEVATED};
        border-color: {TEXT};
    }}

    QPushButton#primaryButton:hover {{
        background: #30332f;
        border-color: #30332f;
    }}

    QPushButton#secondaryButton {{
        background: {SURFACE_ELEVATED};
        border-color: {BORDER_STRONG};
    }}

    QPushButton#ghostButton, QPushButton#settingsButton {{
        background: transparent;
        border-color: {BORDER};
        color: {TEXT_MUTED};
    }}

    QPushButton#ghostButton:hover, QPushButton#settingsButton:hover {{
        color: {TEXT};
        background: {SURFACE};
        border-color: {ACCENT};
    }}

    QSplitter::handle {{
        background: transparent;
        margin: 6px;
    }}

    QScrollArea#previewCanvas {{
        background: {CANVAS};
        border: 1px solid #2a2d30;
        border-radius: 16px;
    }}

    QScrollArea#previewCanvas QLabel {{
        color: #d7d1c5;
        background: transparent;
    }}

    QTableWidget {{
        background: {SURFACE_ELEVATED};
        alternate-background-color: {SURFACE};
        color: {TEXT};
        border: 1px solid {BORDER};
        border-radius: 14px;
        gridline-color: transparent;
        selection-background-color: {SELECTION};
        selection-color: {TEXT};
    }}

    QTableWidget::item {{
        padding: 8px 10px;
        border-bottom: 1px solid #eee5d8;
    }}

    QHeaderView::section {{
        background: {SURFACE};
        color: {TEXT_MUTED};
        border: none;
        border-bottom: 1px solid {BORDER};
        padding: 9px 10px;
        font-weight: 700;
    }}

    QDialogButtonBox QPushButton {{
        min-width: 84px;
    }}
    """
