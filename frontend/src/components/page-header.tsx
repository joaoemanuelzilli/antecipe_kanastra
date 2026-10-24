import type { ReactNode } from 'react';
import { RoleSwitch } from './shell';
import './page-header.css';

export interface PageHeaderProps {
  title: string;
  description: ReactNode;
}

export function PageHeader({ title, description }: PageHeaderProps) {
  return <header className="page-header"><div className="page-header-copy"><h1>{title}</h1><p>{description}</p></div><RoleSwitch /></header>;
}
