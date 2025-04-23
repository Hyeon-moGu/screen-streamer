> **rtsp-screen-streamer**  
> 실시간 화면을 MJPEG 포맷으로 캡처해 RTSP로 전송하는 Go 기반 화면 스트리머

---
<p align="right">
  📘 <a href="#english-summary">View English Summary</a>
</p>

## 특징

- RTSP 서버 직접 구현 (`bluenviron/gortsplib` 기반)
- `screenshot`을 이용한 화면 캡처
- `disintegration/imaging` 기반 JPEG 인코딩
- `sync.Pool`, goroutine 워커풀 활용 최적화
- `config.yaml`을 통한 해상도, 프레임레이트, 캡처 모니터 등 설정 제어
- 클라이언트가 연결된 경우만 캡처 및 전송

---

## 설치 및 실행

```bash
# 의존성 설치
go mod tidy

# 실행
go run ./cmd/main.go
```

### 테스트 방법
```bash
# VLC 또는 ffplay로 재생 가능
$ ffplay rtsp://localhost:1554/temp
```

---

## 주요 디렉토리 구조
```
rtsp-screen-streamer/
├── cmd/                  # main entry
├── internal/
│   ├── capture/          # 화면 캡처 (screenshot)
│   ├── encode/           # JPEG 인코딩 + 워커풀
│   └── stream/           # RTSP 서버 및 전송 처리
├── pkg/config/           # 설정 로더 (viper)
├── config.yaml           # 외부 설정 파일
├── go.mod / go.sum
└── README.md
```

---

## 개선 방향
- H.264 전송 지원 (FFmpeg)
- GUI 지원 및 멀티 모니터 실시간 선택

---
## English Summary

rtsp-screen-streamer is a lightweight RTSP screen streamer written in Go.

- Captures screen as MJPEG using `screenshot` and `imaging`
- Streams via built-in RTSP server (`gortsplib`)
- Starts only when a client connects
- Configurable via `config.yaml` (resolution, fps, monitor)
