from __future__ import annotations

import asyncio
import os
import sys

from PySide6.QtWidgets import QApplication

from image_generator.app import ConfigurationCancelled, ImageGeneratorApplication
from image_generator.ui.theme import apply_app_theme


def main() -> int:
    import qasync

    os.environ.setdefault("QT_ENABLE_HIGHDPI_SCALING", "1")
    app = QApplication(sys.argv)
    apply_app_theme(app)
    loop = qasync.QEventLoop(app)
    asyncio.set_event_loop(loop)

    image_app = ImageGeneratorApplication()
    try:
        window = image_app.create_window()
    except ConfigurationCancelled:
        return 0
    window.show()
    app.aboutToQuit.connect(loop.stop)

    with loop:
        loop.run_forever()
    return 0
