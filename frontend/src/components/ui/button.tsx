import { Slot } from '@radix-ui/react-slot';
import type { ButtonHTMLAttributes } from 'react';
import { cx } from '@/lib/format';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'outline' | 'ghost' | 'danger';
  size?: 'default' | 'small' | 'icon';
  asChild?: boolean;
}
export function Button({ className, variant = 'primary', size = 'default', asChild, ...props }: ButtonProps) {
  const Component = asChild ? Slot : 'button';
  return <Component className={cx('button', `button-${variant}`, `button-${size}`, className)} {...props} />;
}
