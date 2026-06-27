def test_package_imports() -> None:
    import image_generator

    assert image_generator.__version__ == "0.1.0"
