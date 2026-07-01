from image_generator.app import ImageGeneratorApplication
from image_generator.config import CONFIG_DIR_ENV
from image_generator.models import GenerationParams


def test_missing_first_run_settings_do_not_block_startup(monkeypatch, tmp_path, qtbot) -> None:
    monkeypatch.setenv(CONFIG_DIR_ENV, str(tmp_path))
    image_app = ImageGeneratorApplication()

    window = image_app.create_window()
    qtbot.addWidget(window)

    assert image_app.window is window
    assert image_app.queue is not None
    assert window.windowTitle() == "图片生成器"


def test_submit_task_with_missing_settings_warns_without_queueing(
    monkeypatch,
    tmp_path,
    qtbot,
) -> None:
    monkeypatch.setenv(CONFIG_DIR_ENV, str(tmp_path))
    image_app = ImageGeneratorApplication()
    window = image_app.create_window()
    qtbot.addWidget(window)
    assert image_app.queue is not None

    warnings: list[tuple[str, str]] = []
    monkeypatch.setattr(
        "image_generator.app.QMessageBox.warning",
        lambda _parent, title, message: warnings.append((title, message)),
    )

    image_app.submit_task(GenerationParams(prompt="一只狐狸"))

    assert warnings == [
        (
            "设置不完整",
            "请先在“设置”中补全以下内容：\n接口地址不能为空\nAPI 密钥不能为空",
        )
    ]
    assert image_app.queue.tasks == {}
