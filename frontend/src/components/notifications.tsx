import { Toaster, toast } from 'sonner';
import { CheckCircle2, CircleAlert, Info, X } from 'lucide-react';
import './notifications.css';

export const notify = {
  success: (message: string, description?: string) => toast.success(message, { description, duration: 5000 }),
  error: (message: string, description?: string) => toast.error(message, { description, duration: 8000 }),
  info: (message: string, description?: string) => toast.info(message, { description, duration: 6000 }),
};
export function Notifications() {
  return <Toaster position="top-center" closeButton visibleToasts={3} gap={12} offset={24} mobileOffset={16} theme="light" icons={{ success: <CheckCircle2 size={21} />, error: <CircleAlert size={21} />, info: <Info size={21} />, close: <X size={16} /> }} toastOptions={{ className: 'system-notification', closeButtonAriaLabel: 'Fechar notificação' }} containerAriaLabel="Notificações do sistema" />;
}
