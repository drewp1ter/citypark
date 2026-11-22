#!/usr/bin/env python3
"""Check root metadata layout and known legacy XML color-import failures.

Run before packaging or loading XML. This checks layout and metadata identities;
it does not replace Designer compilation or runtime validation.
"""
import argparse
from pathlib import Path
import re
import sys
import xml.etree.ElementTree as ET

MD_NS = 'http://v8.1c.ru/8.3/MDClasses'
NS = {'md': MD_NS}
IRREGULAR_FOLDERS = {
    'FilterCriterion': 'FilterCriteria',
    'ChartOfAccounts': 'ChartsOfAccounts',
    'ChartOfCalculationTypes': 'ChartsOfCalculationTypes',
    'ChartOfCharacteristicTypes': 'ChartsOfCharacteristicTypes',
}


def check_custom_color_references(root):
    """Catch custom Color references rejected by the owner's 8.3.11 importer.

    Built-in style references are retained. This is a project compatibility
    check based on observed import failures, not a general XDTO validator.
    """
    colors = set()
    for path in (root / 'StyleItems').glob('*.xml'):
        properties = ET.parse(path).find('md:StyleItem/md:Properties', NS)
        if properties is not None and properties.findtext('md:Type', namespaces=NS) == 'Color':
            colors.add(path.stem)
    errors = []
    for path in root.rglob('*.xml'):
        if '.codex' in path.relative_to(root).parts:
            continue
        text = path.read_text(encoding='utf-8-sig')
        references = sorted(set(re.findall(r'\bstyle:(\w+)', text)) & colors)
        if references:
            errors.append(f'{path.relative_to(root)}: custom color references rejected by '
                          f'the legacy XML importer: {", ".join(references)}')
    return errors


def check_export(root):
    errors = []
    configuration = ET.parse(root / 'Configuration.xml').find('md:Configuration', NS)
    if configuration is None:
        return 0, ['Configuration.xml does not declare a Configuration object']
    children = configuration.find('md:ChildObjects', NS)
    if children is None:
        return 0, ['Configuration.xml has no ChildObjects list']
    count = 0
    for declaration in children:
        count += 1
        kind = declaration.tag.split('}')[-1]
        name = declaration.text or ''
        folder = IRREGULAR_FOLDERS.get(kind, kind + 's')
        path = root / folder / (name + '.xml')
        if not path.is_file():
            misplaced = root / '.codex' / folder / (name + '.xml')
            hint = f' (misplaced under .codex/{folder})' if misplaced.is_file() else ''
            errors.append(f'Missing {folder}/{name}.xml{hint}')
            continue
        try:
            metadata = ET.parse(path).getroot().find('md:' + kind, NS)
            actual_name = None if metadata is None else metadata.findtext('md:Properties/md:Name', namespaces=NS)
            if metadata is None or actual_name != name:
                errors.append(f'{folder}/{name}.xml does not declare {kind}.{name}')
        except ET.ParseError as exc:
            errors.append(f'Invalid XML in {folder}/{name}.xml: {exc}')
    errors.extend(check_custom_color_references(root))
    return count, errors


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source', nargs='?', type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args()
    try:
        count, errors = check_export(args.source)
    except (OSError, ET.ParseError) as exc:
        print(f'FAIL: {exc}', file=sys.stderr)
        return 1
    for error in errors:
        print('FAIL: ' + error, file=sys.stderr)
    if errors:
        print(f'{len(errors)} errors in {count} root metadata declarations.', file=sys.stderr)
        return 1
    print(f'PASS: all {count} root metadata declarations have correctly named XML files; '
          'legacy custom-color reference scan is clean.')
    return 0


if __name__ == '__main__':
    sys.exit(main())
