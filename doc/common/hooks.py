import re

def replace_specific_links(markdown, *, page, config, files, **kwargs):
    # Replace link from /CHANGELOG.md to page with changelog
    markdown = re.sub(
        r'\[([^]]+)](\(|:\s*).*CHANGELOG.md(\)|\s*)',
        r'[\1]\2changelog.md\3',
        markdown
    )

    # Replace link from /LICENSE to page with license
    markdown = re.sub(
        r'\[([^]]+)](\(|:\s*).*LICENSE(\)|\s*)',
        r'[\1]\2license.md\3',
        markdown
    )

    # Replace links for assets
    markdown = re.sub(
        r'\[([^]]+)](\(|:\s*)(\.\./){2}(asset/.+)(\)|\s*)',
        rf'[\1]\2https://raw.githubusercontent.com/{config.repo_name}/refs/heads/master/\4\5',
        markdown
    )

    # Replace links for code
    markdown = re.sub(
        r'\[([^]]+)](\(|:\s*)(\.\./){2}(.+)(\)|\s*)',
        rf'[\1]\2{config.repo_url}/tree/master/\4\5',
        markdown
    )

    return markdown
