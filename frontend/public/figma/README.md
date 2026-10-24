# Assets do Figma do time

Arquivo EFdeJ9T6mnBayCJcbh60aT, seção1:913. Os arquivos abaixo foram obtidos pelos exports do plugin oficial, sem usar screenshots de telas como assets de implementação. As URLs temporárias não ficam no código.

| Arquivo | Nó/uso |
|---|---|
| login-background.png | 1:364, imagem de fundo original |
| logo-white.svg | 1:390, marca no login296×48 |
| mascot-welcome.png | 1:422, mascote no convite |
| eye-closed.svg | 1:375, visibilidade de senha24×24 |
| arrow-right.svg | 1:417, ação do convite24×24 |
| logo-mark.svg | 1:453, marca na navegação41×32 |
| house.svg | 1:458, dashboard24×24 |
| file-text.svg | 1:466, lotes24×24 |
| sliders.svg | 1:486, regras24×24 |
| hero-shape.svg | 1:580, forma159×188 |
| mascot-magnify.png | 1:577, mascote com recorte211×250 |
| plus.svg | 1:560, novo envio24×24 |
| caret-right.svg | 1:646, abrir lote24×24 |
| notepad.svg | 1:654, total18×18 |
| check-circle.svg | 1:659, aprovadas18×18 |
| x-circle.svg | 1:663, recusadas18×18 |
| assistant-avatar.png | 1:563, export isolado do avatar com sombra |

O export raster inicial do avatar omitiu sua imagem interna. Foi substituído pelo export isolado do próprio nó1:563, que preserva o personagem e a sombra. Nenhum asset foi redesenhado.


## Seção Novo lote1:1365

| Arquivo | Nó/uso |
|---|---|
| new-lot-close.svg | 1:1139, fechar32×32 |
| new-lot-info.svg | 1:1194, abrir instruções24×24 |
| new-lot-collapse.svg | 1:1363, recolher instruções24×24 |
| new-lot-calendar.svg | 1:1151, data24×24 |
| new-lot-cloud.svg | 1:1155, upload32×32 |
| new-lot-sparkle.svg | 1:1161, enviar24×24 |
| mascot-empty-box.png | 1:1624, export isolado110×124 |

O export isolado do mascote vazio foi usado para conservar o recorte do Figma; a imagem original tem proporção diferente. Os SVGs mantêm as dimensões intrínsecas.


## Vídeo do botão de ajuda

`assistant-head.mp4` vem do vídeo de 10 segundos `Robot_head_changing_facial_expre…_20261024112027.mp4`, fornecido pelo usuário em24/10/2026. Versão H.264 de640×360, sem faixa de áudio, com faststart para web. `assistant-head-poster.jpg` é um frame do próprio vídeo para o carregamento inicial e navegadores sem reprodução automática. O botão mantém o recorte circular e abre a orientação local; o vídeo pausa com movimento reduzido, aba oculta ou ajuda aberta.


## Loading do documento — Figma1:1631

`document-review.mp4` usa o vídeo `Robot_reviewing_papers_20261024124729.mp4` fornecido pelo usuário, otimizado para H.264832×468,10segundos, sem áudio e com faststart (cerca de200KB). `document-review-poster.jpg` é um frame do próprio vídeo. Painel480px com vídeo416×234 no desktop; proporção16:9 completa, sem recorte. O ícone de fechar32×32 é o mesmo asset original `new-lot-close.svg`.


### Loop de ida e volta do mascote

`assistant-head-loop.mp4` é a versão usada pelo botão de ajuda:240quadros originais seguidos pelos238quadros internos em ordem inversa, a24fps (19,9167s). Os extremos não se repetem na virada nem no reinício. H.264640×360 sem áudio, faststart, cerca de383KB. A inversão fica no próprio arquivo para funcionar em navegadores sem suporte a velocidade negativa; o loop nativo e a pausa por movimento reduzido permanecem.

- `mascot-empty-box-transparent.png`: versão com alpha do Figma EFdeJ9T6mnBayCJcbh60aT, nó1:1624, exportada em24/10/2026. Substitui a versão com fundo nos estados vazios de Dashboard e Meus lotes, preservando a geometria dos componentes.
