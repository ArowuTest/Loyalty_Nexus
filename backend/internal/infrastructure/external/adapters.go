package external

import (
	"context"
	"time"
)

// VTPassClient provisions airtime and data bundles via VTPass.
// All operations accept an idempotency reference.
type VTPassClient interface {
	TopUpAirtime(ctx context.Context, phone, network string, amountNaira float64, ref string) (vtRef string, err error)
	TopUpData(ctx context.Context, phone, network string, dataMB float64, ref string) (vtRef string, err error)
	VerifyService(ctx context.Context, serviceID string) (bool, error)
}

// MoMoPayer sends money via MTN MoMo Disbursement API.
type MoMoPayer interface {
	Disburse(ctx context.Context, phone string, amountNaira int64, ref string) (momoRef string, err error)
	VerifyAccount(ctx context.Context, phone string) (name string, valid bool, err error)
	GetTransactionStatus(ctx context.Context, momoRef string) (status string, err error)
}

// WalletPassport generates Apple and Google Wallet passes.
type WalletPassport interface {
	IssueApplePass(ctx context.Context, userID string, points int64) (downloadURL string, err error)
	IssueGooglePass(ctx context.Context, userID string, points int64) (saveURL string, err error)
	PushUpdate(ctx context.Context, userID string, points int64) error
}

// ImageGenerator handles AI image generation (HuggingFace, FAL.AI).
type ImageGenerator interface {
	GenerateImage(ctx context.Context, prompt string, model string) (imageURL string, err error)
	RemoveBackground(ctx context.Context, imageURL string) (resultURL string, err error)
	AnimateImage(ctx context.Context, imageURL string) (videoURL string, err error)
}

// KnowledgeGenerator handles knowledge tool generation (Gemini-powered).
type KnowledgeGenerator interface {
	Generate(ctx context.Context, topic string, toolType string, sources []string) (jobID string, err error)
	PollStatus(ctx context.Context, jobID string) (ready bool, outputURL string, err error)
}

// AudioGenerator handles music (Mubert) and TTS (Google Cloud TTS / ElevenLabs).
type AudioGenerator interface {
	GenerateMusic(ctx context.Context, prompt string, durationSecs int) (audioURL string, err error)
	TextToSpeech(ctx context.Context, text string, languageCode string, voice string) (audioURL string, err error)
}

// DocumentProcessor handles transcription (AssemblyAI / Groq Whisper) and translation (Google Translate).
type DocumentProcessor interface {
	TranscribeAudio(ctx context.Context, audioURL string) (transcript string, err error)
	Translate(ctx context.Context, text, targetLanguage string) (translated string, err error)
}

// ObjectStore is the provider-agnostic storage interface used throughout the
// application.  Concrete implementations live in asset_storage.go:
//   - s3Storage      (STORAGE_BACKEND=s3  or auto-detected via AWS_S3_BUCKET)
//   - gcsStorage     (STORAGE_BACKEND=gcs or auto-detected via GCS_BUCKET)
//   - localStorage   (STORAGE_BACKEND=local, or fallback when no cloud creds set)
//
// Use NewAssetStorageFromEnv() to obtain the correct implementation at startup.
// The active backend is logged at startup; switching is a single env-var change.
type ObjectStore = AssetStorage // canonical alias — prefer ObjectStore in new code

// S3Uploader is kept for backwards compatibility with code written before the
// provider-agnostic layer was introduced.  New code should depend on ObjectStore.
//
// Deprecated: Use ObjectStore / AssetStorage instead.
type S3Uploader interface {
	Upload(ctx context.Context, key string, data []byte, contentType string) (presignedURL string, err error)
	GeneratePresignedURL(ctx context.Context, key string, expiresInSeconds int) (url string, err error)
	Delete(ctx context.Context, key string) error
}

// ensure AwsS3Uploader still satisfies the legacy interface
var _ S3Uploader = (*AwsS3Uploader)(nil)

// ensure all AssetStorage backends are compile-time checked
var _ AssetStorage = (*s3Storage)(nil)
var _ AssetStorage = (*gcsStorage)(nil)
var _ AssetStorage = (*localStorage)(nil)

// Ensure time is used (imported for AssetStorage.GeneratePresignedURL signature)
var _ = time.Second
