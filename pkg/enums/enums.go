package enums

type AttachmentTypeEnum string

const (
	AttachmentTypeImage AttachmentTypeEnum = "image"
	AttachmentTypeAudio AttachmentTypeEnum = "audio"
	AttachmentTypeVideo AttachmentTypeEnum = "video"
	AttachmentTypeFile  AttachmentTypeEnum = "file"
)

// AttachmentTypeValues lists every AttachmentTypeEnum; used to create the Postgres enum type.
var AttachmentTypeValues = []string{
	string(AttachmentTypeImage),
	string(AttachmentTypeAudio),
	string(AttachmentTypeVideo),
	string(AttachmentTypeFile),
}

type DeviceOSEnum string

const (
	DeviceOsIOS     = "IOS"
	DeviceOsAndroid = "ANDROID"
	DeviceOsWindow  = "WINDOW"
	DeviceOsLinux   = "LINUX"
	DeviceOsMac     = "MAC"
)

// CategoryIconEnum is a lucide icon name the frontend can draw for an expense category.
type CategoryIconEnum string

var CategoryIconValues = []string{
	"utensils", "coffee", "bike", "car", "bus", "home", "wifi", "smartphone", "users", "baby", "shopping-bag", "shirt",
	"heart-pulse", "pill", "book-open", "graduation-cap", "gift", "plane", "gamepad-2", "dumbbell", "dog", "receipt",
	"piggy-bank", "ellipsis",
}

// CategoryColorEnum maps to a `--category-<color>` theme token in the frontend.
type CategoryColorEnum string

var CategoryColorValues = []string{
	"chart-1", "chart-2", "chart-3", "chart-4", "chart-5",
	"red", "orange", "amber", "green", "teal", "blue", "violet", "pink", "slate",
}
