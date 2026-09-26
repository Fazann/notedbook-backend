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
