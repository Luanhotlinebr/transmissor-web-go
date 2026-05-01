# 📡 Transmissor de Tela via WebSocket (Go)

Aplicação em Go que transmite a tela do computador em tempo real pela rede local utilizando WebSocket.

## ✨ Funcionalidades

- 📺 Visualizar a tela via navegador
- 🎛️ Controlar início/parada da transmissão
- 👀 Monitorar espectadores conectados
- ⚡ Baixa latência com envio direto de frames (JPEG)

---

## 🚀 Como rodar o projeto

### 1. Pré-requisitos

- Go instalado (versão 1.20+ recomendada)

Verifique:
```bash
go version
```
2. Clonar o projeto
```bash
git clone https://github.com/Luanhotlinebr/transmissor-web-go.git
cd transmissor-web-go
```
3. Instalar dependências
```bash
go mod tidy
```
4. Executar o projeto
```bash
go run main.go
```
O sistema abrirá automaticamente no navegador: http://localhost:8080

🖥️ Como compilar e executar (Windows e macOS)
Se você desejar gerar um arquivo executável do projeto para rodar diretamente sem precisar do comando go run, siga as instruções abaixo para o seu sistema:

📦 Windows
Para compilar e gerar o executável:

```bash
GOOS=windows GOARCH=amd64 go build -o transmissor.exe
```
Como executar:
Basta dar um duplo clique no arquivo transmissor.exe gerado na pasta do projeto, ou rodar no terminal:

```dos
transmissor.exe
```
🍎 macOS Para compilar e gerar o executável em Macs com processador Intel:
```bash
GOOS=darwin GOARCH=amd64 go build -o transmissor
```
Para compilar em Macs com processador Apple Silicon (M1/M2/M3):
```bash
GOOS=darwin GOARCH=arm64 go build -o transmissor
```
Como executar:
No terminal, dê permissão de execução (se necessário) e rode o arquivo:

```bash
chmod +x transmissor
./transmissor
```

🐧 Linux
Para compilar e gerar o executável:

```bash
GOOS=linux GOARCH=amd64 go build -o transmissor
```
Como executar:
```bash
./transmissor
```

⚙️ Como funciona (resumo técnico)
Captura da tela usando github.com/kbinani/screenshot.

Compressão em JPEG para otimizar envio.

Transmissão via WebSocket (gorilla/websocket).

Atualização contínua de frames (~30 FPS).

Controle de estado com mutex (concorrência segura).

🔒 Segurança
Apenas o localhost pode:

Iniciar/parar transmissão.

Ver lista de espectadores.

A transmissão é acessível apenas na rede local.

📌 Observações
Funciona melhor em redes locais (LAN).

Pode consumir CPU dependendo da resolução da tela.

Qualidade da imagem ajustada para equilíbrio entre desempenho e clareza.

👨‍💻 Autor
Desenvolvido por Luan Sousa
🔗 https://github.com/Luanhotlinebr

