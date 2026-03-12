package manager

type Instance struct {
	PixelStreamingID         string   `json:"pixelStreamingId"`
	PixelStreamingIP         string   `json:"pixelStreamingIp,omitempty"`
	Host                     string   `json:"host"`
	Port                     int      `json:"port"`
	PixelStreamingServerPort int      `json:"pixelStreamingServerPort,omitempty"`
	PID                      int      `json:"pid"`
	StartTime                int64    `json:"startTime"`
	Model                    string   `json:"model"`
	ExecutablePath           string   `json:"executablePath,omitempty"`
	Args                     []string `json:"args,omitempty"`
	UserID                   string   `json:"userId,omitempty"`
	Subscribed               bool     `json:"subscribed,omitempty"`
	LastSubscribed           string   `json:"lastSubscribed,omitempty"`
}

type StartInstanceRequest struct {
	PixelStreamingServerPort   int    `json:"pixelStreamingServerPort"`
	Model                      string `json:"model"`
	EncoderCodec               string `json:"encoderCodec"`
	EncoderMinQuality          *int   `json:"encoderMinQuality,omitempty"`
	EncoderMaxQuality          *int   `json:"encoderMaxQuality,omitempty"`
	WebRTCMinBitrateMbps       *int   `json:"webrtcMinBitrateMbps,omitempty"`
	WebRTCStartBitrateMbps     *int   `json:"webrtcStartBitrateMbps,omitempty"`
	WebRTCMaxBitrateMbps       *int   `json:"webrtcMaxBitrateMbps,omitempty"`
	PixelStreamingHUDStats     *bool  `json:"pixelStreamingHudStats,omitempty"`
	StdOut                     *bool  `json:"stdOut,omitempty"`
	FullStdOutLogOutput        *bool  `json:"fullStdOutLogOutput,omitempty"`
	WebRTCDisableReceiveAudio  *bool  `json:"webrtcDisableReceiveAudio,omitempty"`
	WebRTCDisableTransmitAudio *bool  `json:"webrtcDisableTransmitAudio,omitempty"`
	D3DRenderer                string `json:"d3dRenderer,omitempty"`
	D3DDebug                   *bool  `json:"d3dDebug,omitempty"`
	NoCheckOther               bool   `json:"noCheckOther"`
	ResX                       int    `json:"resX"`
	ResY                       int    `json:"resY"`
	PixelStreamingID           string `json:"pixelStreamingId"`
	UserID                     string `json:"userId"`
}

type StartInstanceResponse struct {
	Message                  string `json:"message"`
	PixelStreamingID         string `json:"pixelStreamingId,omitempty"`
	PixelStreamingIP         string `json:"pixelStreamingIp,omitempty"`
	PixelStreamingServerPort int    `json:"pixelStreamingServerPort,omitempty"`
	PID                      int    `json:"pid,omitempty"`
	Model                    string `json:"model,omitempty"`
	Reused                   bool   `json:"reused"`
	Error                    string `json:"error,omitempty"`
}

type ModelRequest struct {
	Name    string `json:"name"`
	ExePath string `json:"exePath"`
}

const (
	BuildStatusQueued                = "queued"
	BuildStatusExtractingAndScanning = "extracting_and_scanning"
	BuildStatusReady                 = "ready"
	BuildStatusFailed                = "failed"
)

type Build struct {
	ID           string   `json:"id"`
	FileName     string   `json:"fileName"`
	FileSize     int64    `json:"fileSize"`
	Status       string   `json:"status"`
	Message      string   `json:"message"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
	ZipPath      string   `json:"zipPath"`
	ExtractedDir string   `json:"extractedDir"`
	Executables  []string `json:"executables"`
}
