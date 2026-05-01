package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kbinani/screenshot"
)

var (
	mutex       sync.RWMutex
	lastFrame   []byte
	frameID     int64 // Rastreia o número do frame para evitar envios duplicados
	currentFPS  int
	isStreaming bool 

	viewersMutex sync.Mutex
	viewers      = make(map[string]*Viewer)

	// Configuração do WebSocket
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Permite conexões de qualquer IP na rede local
		},
	}
)

type Viewer struct {
	IP       string    `json:"ip"`
	Focused  bool      `json:"focused"`
	LastSeen time.Time `json:"-"`
}

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	// Garante que apenas o host local (você) possa desligar o servidor
	if !isLocalhost(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<h3>Servidor encerrado com sucesso! Pode fechar esta aba.</h3>"))

	// Cria uma goroutine para dar tempo de enviar a resposta HTTP antes de matar o processo
	go func() {
		time.Sleep(500 * time.Millisecond)
		fmt.Println("🛑 Servidor desligado pelo painel de controle.")
		os.Exit(0) // 0 significa saída sem erros
	}()
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	}
	if err != nil {
		log.Println("Não foi possível abrir o navegador automaticamente:", err)
	}
}

func isLocalhost(r *http.Request) bool {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	return ip == "127.0.0.1" || ip == "::1"
}

func captureLoop() {
	ticker := time.NewTicker(33 * time.Millisecond) // Taxa de quadros da transmissão
	defer ticker.Stop()
	frameCount := 0
	lastTime := time.Now()

	for range ticker.C {
		mutex.RLock()
		streaming := isStreaming
		mutex.RUnlock()

		if !streaming {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		bounds := screenshot.GetDisplayBounds(0)
		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			continue
		}

		var buf bytes.Buffer
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 60})
		if err != nil {
			continue
		}

		mutex.Lock()
		lastFrame = buf.Bytes()
		frameID++ // Incrementa o ID para avisar que há uma imagem nova
		mutex.Unlock()

		frameCount++
		if time.Since(lastTime) >= time.Second {
			mutex.Lock()
			currentFPS = frameCount
			mutex.Unlock()
			frameCount = 0
			lastTime = time.Now()
		}
	}
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	if !isLocalhost(r) {
		http.Redirect(w, r, "/watch", http.StatusSeeOther)
		return
	}

	ip := getLocalIP()
	html := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="pt-BR">
	<head>
		<meta charset="UTF-8">
		<title>Painel de Controle</title>
		<style>
			body { background-color: #1e1e1e; color: white; font-family: Arial; text-align: center; padding: 20px; }
			.container { display: flex; justify-content: center; gap: 20px; flex-wrap: wrap; }
			.box { background: #2d2d2d; padding: 30px; border-radius: 10px; width: 400px; box-shadow: 0 4px 15px rgba(0,0,0,0.5); margin-bottom: 20px; }
			input { width: 90%%; padding: 10px; margin-top: 10px; text-align: center; font-size: 16px; border-radius: 5px; border: none; }
			button { background: #4CAF50; color: white; border: none; padding: 15px; font-size: 18px; border-radius: 5px; cursor: pointer; margin-top: 20px; width: 100%%; font-weight: bold;}
			button.stop { background: #f44336; }
			.viewer-list { text-align: left; margin-top: 15px; background: #1e1e1e; padding: 10px; border-radius: 5px; min-height: 100px; }
			.viewer { padding: 8px; border-bottom: 1px solid #444; font-size: 14px; }
			.focused { color: #4CAF50; }
			.distracted { color: #f44336; }
		</style>
	</head>
	<body>
		<h2>📡 Transmissor de Tela </h2>
		
		<div class="container">
			<div class="box">
				<h3>Controle</h3>
				<p>Compartilhe este link na rede:</p>
				<input type="text" value="http://%s:8080/watch" readonly onclick="this.select()">
				
				<form method="POST" action="/toggle">
					<button type="submit" id="btn">Iniciar Transmissão</button>
				</form>
				<form method="POST" action="/shutdown" onsubmit="return confirm('Tem certeza que deseja encerrar o servidor completamente?');">
          <button type="submit" style="background: #555; margin-top: 10px;">Desligar Servidor</button>
        </form>
			</div>

			<div class="box">
				<h3>Espectadores Conectados</h3>
				<div class="viewer-list" id="viewers">Carregando...</div>
			</div>
		</div>

		<script>
			fetch('/status').then(r => r.text()).then(status => {
				const btn = document.getElementById('btn');
				if (status === "true") {
					btn.innerText = "Parar Transmissão";
					btn.classList.add("stop");
				}
			});

			setInterval(() => {
				fetch('/api/viewers')
					.then(r => r.json())
					.then(data => {
						const list = document.getElementById('viewers');
						list.innerHTML = "";
						if (data.length === 0) {
							list.innerHTML = "<p style='color: #888;'>Ninguém assistindo no momento.</p>";
							return;
						}
						data.forEach(v => {
							const status = v.focused ? "<span class='focused'>[Aba Ativa]</span>" : "<span class='distracted'>[Em outra aba]</span>";
							list.innerHTML += '<div class="viewer">👤 IP: ' + v.ip + ' ' + status + '</div>';
						});
					});
			}, 2000);
		</script>
		<footer><p>Feito por <a href="https://github.com/Luanhotlinebr" target="_blank">Luan Sousa</a></p></footer>
	</body>
	</html>
	`, ip)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func toggleHandler(w http.ResponseWriter, r *http.Request) {
	if !isLocalhost(r) {
		http.Redirect(w, r, "/watch", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		mutex.Lock()
		isStreaming = !isStreaming
		mutex.Unlock()
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	streaming := isStreaming
	mutex.RUnlock()
	fmt.Fprintf(w, "%t", streaming)
}

// NOVO: O cliente agora consome a imagem via WebSocket usando blobs de memória
func watchHandler(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html lang="pt-BR">
	<head><style>body { background: black; color: white; text-align: center; margin: 0; padding: 10px; font-family: Arial;}</style></head>
	<body>
		
		<img id="screen" style="max-width: 100%; border-radius: 8px; box-shadow: 0 4px 15px rgba(0,0,0,0.5);">
		
		<script>
			// Atualiza o FPS
			// setInterval(() => {
			// 	fetch('/fps').then(r => r.text()).then(fps => document.getElementById('fps').innerText = fps);
			// }, 1000);

			// Sinal de vida e foco da aba
			setInterval(() => {
				const isFocused = !document.hidden; 
				fetch('/heartbeat?focused=' + isFocused);
			}, 2000);

			// CONEXÃO WEBSOCKET PARA O VÍDEO
			const img = document.getElementById('screen');
			const ws = new WebSocket('ws://' + location.host + '/ws');
			ws.binaryType = 'blob'; // Avisa que receberemos dados binários (imagem)

			ws.onmessage = function(event) {
				// Cria uma URL temporária na memória do navegador para o JPEG recebido
				const url = URL.createObjectURL(event.data);
				img.src = url;
				
				// Limpa a memória assim que a imagem for renderizada para evitar vazamentos (Memory Leak)
				img.onload = () => {
					URL.revokeObjectURL(url);
				};
			};

			ws.onclose = function() {
				console.log("Transmissão encerrada ou conexão perdida.");
			};
		</script>
	</body>
	</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// NOVO: Gerencia a conexão WebSocket e injeta os frames direto na veia do navegador
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Erro ao atualizar para WebSocket:", err)
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()

	var lastSentID int64 // Guarda o ID do último frame enviado para este usuário

	for range ticker.C {
		mutex.RLock()
		frame := lastFrame
		id := frameID
		streaming := isStreaming
		mutex.RUnlock()

		// Se a transmissão parou, não há frame ou o frame é repetido, ignora!
		if !streaming || len(frame) == 0 || id == lastSentID {
			continue
		}

		// Envia a imagem em formato binário direto para o JavaScript
		err = conn.WriteMessage(websocket.BinaryMessage, frame)
		if err != nil {
			break // Se der erro (ex: espectador fechou a aba), quebra o loop e fecha conexão
		}
		
		lastSentID = id
	}
}

func fpsHandler(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	fps := currentFPS
	mutex.RUnlock()
	fmt.Fprintf(w, "%d", fps)
}

func heartbeatHandler(w http.ResponseWriter, r *http.Request) {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	focused := r.URL.Query().Get("focused") == "true"

	viewersMutex.Lock()
	viewers[ip] = &Viewer{IP: ip, Focused: focused, LastSeen: time.Now()}
	viewersMutex.Unlock()

	w.WriteHeader(http.StatusOK)
}

func apiViewersHandler(w http.ResponseWriter, r *http.Request) {
	if !isLocalhost(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	viewersMutex.Lock()
	defer viewersMutex.Unlock()

	activeViewers := []Viewer{}
	for ip, v := range viewers {
		if time.Since(v.LastSeen) < 5*time.Second {
			activeViewers = append(activeViewers, *v)
		} else {
			delete(viewers, ip)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activeViewers)
}

func main() {
	go captureLoop()
	http.HandleFunc("/", adminHandler)
	http.HandleFunc("/toggle", toggleHandler)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/watch", watchHandler)
	http.HandleFunc("/fps", fpsHandler)
	// Rota alterada para o WebSocket
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/shutdown", shutdownHandler)
	http.HandleFunc("/heartbeat", heartbeatHandler)
	http.HandleFunc("/api/viewers", apiViewersHandler)

	go func() {
		time.Sleep(1 * time.Second)
		openBrowser("http://localhost:8080/")
	}()

	fmt.Println("Servidor WebSocket rodando. Verifique o navegador.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}