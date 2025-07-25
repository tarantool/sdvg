import re


def replace_specific_links(markdown, *, page, config, files, **kwargs):
    markdown = re.sub(
        r'\[([^]]+)](\(|:\s*).*CHANGELOG.md(\)|\s*)',
        r'[\1]\2changelog.md\3',
        markdown
    )
    markdown = re.sub(
        r'\[([^]]+)](\(|:\s*).*LICENSE(\)|\s*)',
        r'[\1]\2license.md\3',
        markdown
    )
    markdown = re.sub(
        r'\[([^]]+)](\(|:\s*)(\.\./){2}(.+)(\)|\s*)',
        rf'[\1]\2{config.repo_url}/tree/master/\4\5)',
        markdown
    )

    return markdown
