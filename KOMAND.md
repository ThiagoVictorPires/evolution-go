# Evolution GO: versão Komand

Fork da [Evolution GO](https://github.com/evolution-foundation/evolution-go) 0.7.2 usado pela plataforma Komand Pro para o WhatsApp.
Branch de trabalho: `komand-fix-list`. A `main` deste repositório espelha o upstream.

| | |
|---|---|
| Base | evolution-go 0.7.2 (`9337afc`) |
| whatsmeow | cópia local em `whatsmeow-lib/`, base `b572e5bcb92b` (30/06/2026), com 3 ajustes |
| Versão do fork | arquivo `VERSION` (hoje `0.7.2-komand3`) |
| Implantação | Easypanel, serviço Compose do repositório `evolution-go-easypanel` (privado): constrói esta branch |

> Este arquivo é o registro do que funciona e do que mudou. **Atualize a tabela de status e o registro de mudanças a cada alteração.**

## O que funciona (estado em 27/10/2026)

| Função | Estado | Como foi conferido |
|---|---|---|
| Texto, imagem, vídeo, áudio, documento | Em uso | Operação diária da plataforma |
| **Lista** (`POST /send/list`) | **Funcionando** | Aparece no celular e no WhatsApp Web; o toque volta pelo webhook (`selectedRowID`). Corrigido no commit `8b441ef` |
| **Botões de resposta** (`POST /send/button`, até 3) | **Funcionando** (implantado e testado em 28/10: aparecem, o toque volta citando a pergunta) | Em 28/10 o WhatsApp recusava o `ButtonsMessage` com `server returned error 405` (upstream #170). Agora sai como `InteractiveMessage` native_flow `quick_reply` com o `<biz>` completo; o toque deve voltar como `InteractiveResponseMessage` (`paramsJSON` com o `id`) |
| **Citação com texto** (`quoted.messageId` + `text` + `fromMe`) | **Funcionando** | Resposta citando enviada pela plataforma; o cliente vê a citação com o texto. Commit `0b5da70` |
| Enquete (`POST /send/poll`) | Envio funciona | A enquete aparece. A leitura do voto é um bug conhecido do upstream (`Failed to decrypt vote`, issue #219): **não resolvido** |
| **Reconexão automática ao reiniciar** | **Funcionando** | `CONNECT_ON_STARTUP=true` no compose. Depois do deploy a instância voltou conectada sozinha; sem a opção, ficava desligada até alguém pedir a conexão |
| Pedido de histórico (`POST /chat/history-sync`, "Carregar mensagens anteriores") | Corrigido e implantado, **aguardando teste** | Commit `40102eb`. O teste anterior falhou com `no signal session established`; ainda não repetido |
| Webhook | Funcionando | A URL precisa de `?secret=...`; sem ele a plataforma responde 401. Eventos usados: `MESSAGE`, `CONNECTION`, `QRCODE`, `READ_RECEIPT`, `HISTORY_SYNC` |
| Foto de perfil (`POST /user/avatar`) | Funcionando | Enviar o JID (`5534...@s.whatsapp.net`). Com o número puro e 9º dígito a chamada travava |
| Agenda de contatos (`GET /user/contacts`) | Funcionando | Traz nome salvo, nome do perfil e nome comercial (~2.800 contatos na instância principal) |
| **`@lid` → número** (`POST /user/lid-to-pn`) | **Funcionando** | Rota nossa: recebe `{"lids":[...]}` e devolve os números que o mapa da whatsmeow conhece (só os mapeados). Teste em 28/10: 135 de 135 `@lid` de uma instância ganharam número; com o nome da agenda e do CRM, a plataforma deixou só 1 `@lid` sem número e 32 de 154 contatos sem nome |
| `@lid` | Resolvido na GO | O webhook traz `SenderAlt`/`RecipientAlt`; medido em 27/10: nenhum contato sem número nesta instância. A API só expõe número → LID (`/user/info`, `/user/check`), não o inverso |
| Carrossel, Pix, localização, contato, reação, editar, apagar, presença, etiquetas | Rotas existem | Ainda não usadas nem testadas por nós |
| Fixar, arquivar, silenciar conversa | Não funciona | O próprio upstream marca como `TODO: not working` |

## O que mudamos em relação ao upstream

1. **`whatsmeow-lib/send.go`** (commit `8b441ef`)
   - O nó `<biz><list v="2" type="…"/>` passa a sair com `type="product_list"`, mais os atributos `actual_actors`, `host_storage`, `privacy_mode_ts` e o nó `<quality_control source_type="third_party">`, igual ao cliente oficial (referência: tulir/whatsmeow#1235).
   - Respostas de botão/lista (`ButtonsResponseMessage`, `ListResponseMessage`) deixam de levar `<biz>` (o servidor devolvia erro 479; tulir/whatsmeow#1221).
   - `hasBizNode`: se quem chama já mandou o próprio `<biz>` (os botões da GO), a biblioteca não põe outro.
2. **`pkg/sendMessage/service/send_service.go`**
   - `SendList` manda a `ListMessage` sem o embrulho `DocumentWithCaptionMessage` e sem `<bot>`. Com o formato antigo a API devolvia 200, mas a lista não aparecia (issues #59, #170, #204 do upstream).
   - `SendButton` (só botões de resposta): `InteractiveMessage` native_flow `quick_reply` no topo, sem `DocumentWithCaptionMessage`, sem `<bot>` e sem cabeçalho de mídia, com o `<biz actual_actors host_storage privacy_mode_ts>` + `<interactive native_flow v=1><native_flow v=9 name=mixed/>` + `<quality_control>` (função `komandReplyBizNode`, formato da tulir/whatsmeow#1235 e do fork itsliaaa/baileys). Pix e botões de link/copiar/ligar ficaram como estavam.
   - `QuotedStruct` ganha `text` e `fromMe`; a citação usa o texto e preenche o autor (nosso JID se `fromMe`, senão o do destinatário). Antes a citação chegava vazia (commit `0b5da70`).
3. **`pkg/user/`** (`service`, `handler`) e **`pkg/routes/routes.go`**: rota nova `POST /user/lid-to-pn` (`{"lids": ["123@lid", ...]}` até 500 → `{"data": {"123@lid": "5534...@s.whatsapp.net"}}`). Usa `Store.LIDs.GetPNForLID`; só devolve os mapeados.
4. **`pkg/chat/service/chat_service.go`** (commit `40102eb`): `HistorySyncRequest` usa `SendPeerMessage` (pedido para o próprio celular), não para o chat do contato (upstream #150).
5. **`Dockerfile`** copia `whatsmeow-lib` antes do `go mod download`; **`go.mod`** tem `replace go.mau.fi/whatsmeow => ./whatsmeow-lib`.

## Implantação

1. O repositório `evolution-go-easypanel` tem um `docker-compose.yml` com `build.context` apontando para `…/evolution-go.git#komand-fix-list`.
2. No Easypanel: serviço `evolution-go` → **Deploy**. O primeiro build leva ~2 minutos (compila a GO); com a imagem já pronta, 7 segundos.
3. Variáveis relevantes (no compose): `CONNECT_ON_STARTUP: "true"`, `DATABASE_SAVE_MESSAGES: "false"`. Segredos (`GLOBAL_API_KEY`, `POSTGRES_PASSWORD`) ficam só no painel do Easypanel, nunca neste repositório.

## Como atualizar com o upstream

```bash
git remote add upstream https://github.com/evolution-foundation/evolution-go.git
git fetch upstream
git checkout komand-fix-list
git merge upstream/main     # resolver conflitos mantendo os itens 1 a 4 acima
```

- Se o upstream trocar a versão da whatsmeow no `go.mod`, baixe essa versão para `whatsmeow-lib/` e reaplique o item 1 (funções `getButtonTypeFromMessage`, `getButtonAttributes`, `hasBizNode` e o bloco que monta o `<biz>`).
- Se o upstream corrigir lista, citação ou histórico, prefira a versão dele e remova o ajuste daqui.

## Cuidados e pendências

- **Compilação local:** a GO precisa de compilador C (pacote webp), então não compila numa máquina sem ele. Os ajustes foram conferidos com `go vet` usando um substituto do webp; o primeiro build completo acontece no Easypanel.
- **Voto de enquete:** corrigir a leitura do voto (upstream #219).
- **Capacidade:** cada número novo é mais uma conexão nesta mesma GO. Acompanhar CPU e memória.
- **Reinício:** a GO não avisa quando cai ou reinicia. A plataforma confere a conexão a cada 30 s (`check-whatsapp-connection`, modo `passive`).

## Registro de mudanças

| Data | Commit | Mudança |
|---|---|---|
| 27/10/2026 | `8b441ef` | Lista volta a aparecer; whatsmeow local; `VERSION` 0.7.2-komand1 |
| 27/10/2026 | `0b5da70` | Citação com o texto; `VERSION` 0.7.2-komand2 |
| 27/10/2026 | `40102eb` | Pedido de histórico vai para o próprio celular; `VERSION` 0.7.2-komand3 |
| 27/10/2026 | (compose `10e4f07`) | `CONNECT_ON_STARTUP=true` |
| 28/10/2026 | `fd3b9cf` | Rota `POST /user/lid-to-pn`; `VERSION` 0.7.2-komand4 (implantada e testada) |
| 28/10/2026 | `51f30be` | Botões de resposta como `InteractiveMessage` + `<biz>` completo; `VERSION` 0.7.2-komand5 (implantada e testada) |
