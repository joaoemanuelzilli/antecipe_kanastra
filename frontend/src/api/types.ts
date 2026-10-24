/** Contrato v1.5 do grupo, commit ef95af8. Valores monetários inteiros em centavos. */
export type MetodoPreco='DESCONTO_SIMPLES'|'DESCONTO_COMPOSTO';
export interface Precificacao {metodo:MetodoPreco;taxa_mensal_percentual:number;base_dias:number;descricao:string;fonte_metodo?:string;sobrescritos_pelo_yaml?:string[]}
export interface Regra {codigo:string;ativo:boolean;parametros:Record<string,unknown>}
export interface Regras {fundo:{nome:string;cnpj:string};validadores:Regra[];precificacao:{metodo:MetodoPreco;taxa_mensal_percentual:number;base_dias:number}}
export interface ParametroCatalogo {nome:string;rotulo:string;tipo:'inteiro'|'decimal'|'lista_texto'|'opcao';unidade?:string;minimo?:number;maximo?:number;opcoes?:string[];opcional?:boolean}
export interface RegraCatalogo {codigo:string;titulo:string;descricao:string;parametros:ParametroCatalogo[]}
export interface Config {exigir_login?:boolean;fundo:{nome:string;cnpj:string};data_base_padrao:string;regras_versao:number;precificacao:Precificacao;upload:{tamanho_maximo_bytes:number;separador:string;colunas:string[]}}
export interface ErroDetalhe {linha?:number;campo:string;problema:string}
export interface ApiErro {codigo:string;mensagem:string;detalhes?:ErroDetalhe[]}
export interface Recusa {codigo:string;titulo:string;mensagem:string;acao:string;dados:Record<string,unknown>}
export interface Nota {linha:number;numero:string;tipo:string;cedente:{nome:string;documento:string};sacado:{nome:string;documento:string};vencimento:string|null;dias:number|null;status:'aprovada'|'recusada';face_centavos:number|null;desconto_centavos:number|null;liquido_centavos:number|null;recusas:Recusa[];calculo:{metodo:MetodoPreco;taxa_mensal_percentual:number;base_dias:number;dias:number;texto:string}|null}
export interface Totais {linhas:number;aprovadas:number;recusadas:number;face_centavos:number;desconto_centavos:number;liquido_centavos:number}
export interface MotivoResumo {codigo:string;titulo:string;quantidade:number;acao:string}
export interface Etapa {codigo:'lendo'|'validando'|'concentracao'|'precificando'|'salvando';titulo:string;status:'pendente'|'em_andamento'|'concluida'}
export interface Cessao {id:string;lote_id:string;confirmada_em:string;notas_cedidas:number;face_centavos:number;desconto_centavos:number;liquido_centavos:number}
export interface NotaExcluida {linha:number;codigo:string;titulo:string;mensagem:string}
export interface ResultadoCessao {cessao:Cessao;notas_excluidas:NotaExcluida[]}
export interface Lote {id:string;status:'na_fila'|'processando'|'concluido'|'erro';etapa:Etapa['codigo']|null;etapas:Etapa[];progresso:{linhas_processadas:number;linhas_total:number;percentual:number};posicao_fila:number|null;data_base:string;regras_versao:number;precificacao:Precificacao;criado_em:string;iniciado_em:string|null;concluido_em:string|null;duracao_ms:number|null;totais:Totais|null;motivos:MotivoResumo[];erros_configuracao:{codigo:string;mensagem:string}[];cessao:Cessao|null;erro:ApiErro|null}
export interface UploadResposta {lote_id:string;status:Lote['status'];replay:boolean}
export interface PaginaNotas {pagina:number;por_pagina:number;total:number;total_paginas:number;itens:Nota[]}
export interface FiltrosNotas {pagina?:number;por_pagina?:number;status?:Nota['status'];motivo?:string;busca?:string;ordenar?:'linha'|'vencimento'|'dias'|'face'|'liquido'|'sacado';direcao?:'asc'|'desc'}
export interface RegrasPublicadas {versao:number;publicada_em:string;autor:string;regras:Regras;catalogo:RegraCatalogo[];precificacao_efetiva:Precificacao;catalogo_precificacao:{parametros:ParametroCatalogo[]}}
export interface VersaoRegras {versao:number;autor:string;publicada_em:string;resumo:string[]}
export interface DetalheVersao {versao:number;autor:string;publicada_em:string;regras:Regras}
export interface Previa {lote_id:string;linhas:number;antes:{regras_versao:number;aprovadas:number;recusadas:number;liquido_centavos:number};depois:{aprovadas:number;recusadas:number;liquido_centavos:number};motivos:{codigo:string;titulo:string;antes:number;depois:number}[];mudancas:string[]}
export interface LoteLocal {id:string;arquivo:string;criado_em:string;nome?:string}
