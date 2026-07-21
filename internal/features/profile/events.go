package profile

import (
	"time"

	"github.com/OrisunLabs/go-orisun-datastar/internal/eventstore"
	"github.com/OrisunLabs/go-orisun-datastar/internal/protectedpii"
)

const (
	ProfileBioUpdated          = "ProfileBioUpdated"
	ProfileImageUploaded       = "ProfileImageUploaded"
	ProfileHeaderImageUploaded = "ProfileHeaderImageUploaded"
)

const (
	ProfileBioUpdatedIDField          = "profileBioUpdatedId"
	ProfileBioUpdatedBioField         = "bio"
	ProfileBioUpdatedBioHashField     = "bioHash"
	ProfileBioUpdatedAtField          = "updatedAt"
	ProfileImageUploadedIDField       = "profileImageUploadedId"
	ProfileHeaderImageUploadedIDField = "profileHeaderImageUploadedId"
	ProfileImageURLField              = "imageUrl"
	ProfileImageUploadedAtField       = "uploadedAt"
	ProfileScopeUserRegisteredIDKey   = "userRegisteredId"
	ProfileScopeUserRegisteredIDField = "scope." + ProfileScopeUserRegisteredIDKey
)

type ProfileBioUpdatedEvent struct {
	ProfileBioUpdatedID string             `json:"profileBioUpdatedId"`
	Bio                 protectedpii.Value `json:"bio"`
	BioHash             string             `json:"bioHash"`
	UpdatedAt           string             `json:"updatedAt"`
	Scope               ProfileUserScope   `json:"scope"`
}

type ProfileImageUploadedEvent struct {
	ProfileImageUploadedID string           `json:"profileImageUploadedId"`
	ImageURL               string           `json:"imageUrl"`
	UploadedAt             string           `json:"uploadedAt"`
	Scope                  ProfileUserScope `json:"scope"`
}

type ProfileHeaderImageUploadedEvent struct {
	ProfileHeaderImageUploadedID string           `json:"profileHeaderImageUploadedId"`
	ImageURL                     string           `json:"imageUrl"`
	UploadedAt                   string           `json:"uploadedAt"`
	Scope                        ProfileUserScope `json:"scope"`
}

type ProfileUserScope struct {
	UserRegisteredID string `json:"userRegisteredId"`
}

func NewProfileBioUpdatedEvent(profileBioUpdatedID, bio string, updatedAt time.Time, userRegisteredID string, subjectKey protectedpii.SubjectDataKey, metadata map[string]any) eventstore.DomainEvent {
	protector := protectedpii.FromEnv()
	return eventstore.DomainEvent{
		EventID:   profileBioUpdatedID,
		EventType: ProfileBioUpdated,
		Data: eventstore.MustData(ProfileBioUpdatedEvent{
			ProfileBioUpdatedID: profileBioUpdatedID,
			Bio:                 protector.MustProtectWithDataKey(bio, ProfileBioUpdatedBioField, subjectKey),
			BioHash:             protector.BlindIndex(ProfileBioUpdatedBioField, bio),
			UpdatedAt:           updatedAt.Format(time.RFC3339),
			Scope:               ProfileUserScope{UserRegisteredID: userRegisteredID},
		}),
		Metadata: metadata,
	}
}

func NewProfileImageUploadedEvent(profileImageUploadedID, imageURL string, uploadedAt time.Time, userRegisteredID string, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   profileImageUploadedID,
		EventType: ProfileImageUploaded,
		Data: eventstore.MustData(ProfileImageUploadedEvent{
			ProfileImageUploadedID: profileImageUploadedID,
			ImageURL:               imageURL,
			UploadedAt:             uploadedAt.Format(time.RFC3339),
			Scope:                  ProfileUserScope{UserRegisteredID: userRegisteredID},
		}),
		Metadata: metadata,
	}
}

func NewProfileHeaderImageUploadedEvent(profileHeaderImageUploadedID, imageURL string, uploadedAt time.Time, userRegisteredID string, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   profileHeaderImageUploadedID,
		EventType: ProfileHeaderImageUploaded,
		Data: eventstore.MustData(ProfileHeaderImageUploadedEvent{
			ProfileHeaderImageUploadedID: profileHeaderImageUploadedID,
			ImageURL:                     imageURL,
			UploadedAt:                   uploadedAt.Format(time.RFC3339),
			Scope:                        ProfileUserScope{UserRegisteredID: userRegisteredID},
		}),
		Metadata: metadata,
	}
}

func Channel(userRegisteredID string) string {
	return "profile." + userRegisteredID
}
