"""Configuration file for the Sphinx documentation builder."""

import sys
from pathlib import Path
from subprocess import check_output

import requests

# -- General configuration ------------------------------------------------

project = "dra-usbip-driver"

# Use git describe for versioning
root = Path(__file__).absolute().parent.parent
try:
    release = (
        check_output("git describe --tags --always".split(), cwd=root)
        .decode()
        .strip()
    )
except Exception:
    release = "dev"

if "+" in release or "-" in release:
    # Not on a clean tag, use branch name
    try:
        version = (
            check_output("git branch --show-current".split(), cwd=root)
            .decode()
            .strip()
        )
    except Exception:
        version = "dev"
else:
    version = release

extensions = [
    "sphinx.ext.intersphinx",
    "sphinx_copybutton",
    "sphinx_design",
    "myst_parser",
]

# So we can use the ::: syntax
myst_enable_extensions = ["colon_fence"]

# If true, Sphinx will warn about all references where the target cannot
# be found.
nitpicky = True

# The master toctree document.
master_doc = "index"

# List of patterns, relative to source directory, that match files and
# directories to ignore when looking for source files.
exclude_patterns = ["_build"]

# The name of the Pygments (syntax highlighting) style to use.
pygments_style = "sphinx"

# Set copy-button to ignore bash prompts
copybutton_prompt_text = r"\$ "
copybutton_prompt_is_regexp = True

# -- Options for HTML output -------------------------------------------------

html_theme = "pydata_sphinx_theme"
github_repo = "dra-usbip-driver"
github_user = "DiamondLightSource"
switcher_json = f"https://{github_user}.github.io/{github_repo}/switcher.json"
switcher_exists = requests.get(switcher_json).ok
if not switcher_exists:
    print(
        "*** Can't read version switcher, is GitHub Pages enabled? \n"
        "    Once Docs CI job has successfully run once, set the "
        "Github pages source branch to be 'gh-pages' at:\n"
        f"    https://github.com/{github_user}/{github_repo}/settings/pages",
        file=sys.stderr,
    )

html_theme_options = {
    "logo": {
        "text": project,
    },
    "use_edit_page_button": True,
    "github_url": f"https://github.com/{github_user}/{github_repo}",
    "switcher": {
        "json_url": switcher_json,
        "version_match": version,
    },
    "check_switcher": False,
    "navbar_end": ["theme-switcher", "icon-links", "version-switcher"],
    "navigation_with_keys": False,
}

html_context = {
    "github_user": github_user,
    "github_repo": github_repo,
    "github_version": version,
    "doc_path": "docs",
}

html_show_sphinx = False
html_show_copyright = False
