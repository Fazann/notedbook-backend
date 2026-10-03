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

// CurrencyEnum is an ISO 4217 code. Amounts are stored in minor units: USD cents, KHR riel, MYR sen.
type CurrencyEnum string

const (
	CurrencyEnumUSD    CurrencyEnum = "USD"
	CurrencyEnumKHRiel CurrencyEnum = "KHR"
	CurrencyEnumMYR    CurrencyEnum = "MYR"
)

var CurrencyValues = []string{
	string(CurrencyEnumUSD),
	string(CurrencyEnumKHRiel),
	string(CurrencyEnumMYR),
}

type PlanPriorityEnum string

const (
	PlanPriorityLow    = "LOW"
	PlanPriorityMedium = "MEDIUM"
	PlanPriorityHigh   = "HIGH"
)

type PlanStatusEnum string

const (
	PlanStatusNotStart   = "NOT_START"
	PlanStatusInProgress = "IN_PROGRESS"
	PlanStatusDone       = "DONE"
)

// ScheduleActivityTypeEnum groups a schedule activity for the weekly summary and its color in the UI.
type ScheduleActivityTypeEnum string

const (
	ScheduleActivityTypeMeeting  ScheduleActivityTypeEnum = "MEETING"
	ScheduleActivityTypeLearning ScheduleActivityTypeEnum = "LEARNING"
	ScheduleActivityTypeWork     ScheduleActivityTypeEnum = "WORK"
	ScheduleActivityTypeExercise ScheduleActivityTypeEnum = "EXERCISE"
	ScheduleActivityTypePersonal ScheduleActivityTypeEnum = "PERSONAL"
	ScheduleActivityTypeFamily   ScheduleActivityTypeEnum = "FAMILY"
	ScheduleActivityTypeOther    ScheduleActivityTypeEnum = "OTHER"
)

var ScheduleActivityTypeValues = []string{
	string(ScheduleActivityTypeMeeting),
	string(ScheduleActivityTypeLearning),
	string(ScheduleActivityTypeWork),
	string(ScheduleActivityTypeExercise),
	string(ScheduleActivityTypePersonal),
	string(ScheduleActivityTypeFamily),
	string(ScheduleActivityTypeOther),
}

// RecurrenceKindEnum is how a schedule activity repeats; NONE is a single, one-off occurrence.
type RecurrenceKindEnum string

const (
	RecurrenceKindNone     RecurrenceKindEnum = "NONE"
	RecurrenceKindDaily    RecurrenceKindEnum = "DAILY"
	RecurrenceKindWeekdays RecurrenceKindEnum = "WEEKDAYS"
	RecurrenceKindWeekly   RecurrenceKindEnum = "WEEKLY"
)

var RecurrenceKindValues = []string{
	string(RecurrenceKindNone),
	string(RecurrenceKindDaily),
	string(RecurrenceKindWeekdays),
	string(RecurrenceKindWeekly),
}
