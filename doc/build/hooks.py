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

    # Replace links for images
    markdown = re.sub(
        r'\[([^]]+)](\(|:\s*)(?:\.\./){2}asset/(.+\.(?:png|jpg))(\)|\s*)',
        rf'[\1]\2assets/images/\3\4',
        markdown
    )

    # Replace links for code
    markdown = re.sub(
        r'\[([^]]+)](\(|:\s*)(?:\.\./){2}(?!asset/)(.+)(\)|\s*)',
        rf'[\1]\2{config.repo_url}/tree/master/\3\4',
        markdown
    )

    return markdown
