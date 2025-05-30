import React from 'react';
import { FaCalendarCheck } from 'react-icons/fa';
import { useColorMode } from '@docusaurus/theme-common';

type Props = {
  version: string;
};

export default function AvailableSince({ version }: Props) {
  const { colorMode } = useColorMode();

  const isDark = colorMode === 'dark';

  const style = {
    display: 'inline-flex',
    alignItems: 'center',
    border: `1px solid var(--ifm-color-primary)`,
    color: isDark ? '#fffdf9' : 'var(--ifm-color-primary)',
    padding: '0.2em 0.6em',
    borderRadius: '999px',
    fontSize: '0.8em',
    fontWeight: 500,
    backgroundColor: isDark ? 'var(--ifm-color-primary)' : '#f0fdfa',
    marginBottom:
      'calc(var(--ifm-heading-vertical-rhythm-bottom) * var(--ifm-leading))'
  };

  return (
    <span className="available-since-badge" style={style}>
      <FaCalendarCheck style={{ marginRight: '0.4em' }} />
      Since {version}
    </span>
  );
}
