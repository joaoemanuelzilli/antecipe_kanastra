import type { ReactNode } from 'react';
import { RoleSwitch } from './shell';
import './page-header.css';

export interface PageHeaderProps {
  title: string;
  description: ReactNode;
  showRoleSwitch?: boolean;
}

export function PageHeader({ title, description, showRoleSwitch = false }: PageHeaderProps) {
  return <header className="page-header"><div className="page-header-copy"><h1>{title}</h1><p>{description}</p></div>{showRoleSwitch && <RoleSwitch />}</header>;
}
