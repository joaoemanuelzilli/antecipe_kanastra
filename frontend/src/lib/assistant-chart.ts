import { z } from 'zod';
export const assistantChartSchema = z.object({
  tipo: z.enum(['barras', 'rosca']),
  titulo: z.string().trim().min(1).max(100),
  metrica: z.enum(['status', 'motivos', 'valores']),
  lote_id: z.string().uuid().optional(),
}).strict().refine(value => value.tipo !== 'rosca' || value.metrica !== 'valores', 'Valores de face, desconto e líquido não são partes independentes');
export type AssistantChartSpec = z.infer<typeof assistantChartSchema>;
export function parseAssistantChart(text: string): AssistantChartSpec | null {
  if (text.length > 2000) return null;
  try { const result = assistantChartSchema.safeParse(JSON.parse(text)); return result.success ? result.data : null; } catch { return null; }
}
