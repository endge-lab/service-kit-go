package auth

import (
	"strings"

	serviceerrors "github.com/endge-lab/service-kit-go/pkg/errors"
)

// ContourAccess описывает гранулярный доступ пользователя внутри одного контура.
type ContourAccess struct {
	Contour     string   `json:"contour"`
	Role        string   `json:"role,omitempty"`
	Groups      []string `json:"groups,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// Identity содержит нормализованный auth context, извлеченный из JWT.
type Identity struct {
	AuthUserID  string
	Username    string
	DisplayName string
	Role        string
	Rules       map[string][]string
	Contours    []ContourAccess
	Groups      []string
	Permissions []string
	SessionID   string
	App         string
	Platform    string
	Scope       []string
	ExpiresAt   string
}

// IsOwner показывает, что пользователь имеет глобальный bypass.
func IsOwner(identity *Identity) bool {
	return identity != nil && strings.EqualFold(strings.TrimSpace(identity.Role), "owner")
}

// HasContour проверяет наличие нужного контура.
func HasContour(identity *Identity, contour string) bool {
	if IsOwner(identity) {
		return true
	}
	needle := strings.TrimSpace(contour)
	if needle == "" || identity == nil {
		return false
	}

	for _, item := range identity.Contours {
		if strings.EqualFold(strings.TrimSpace(item.Contour), needle) {
			return true
		}
	}

	return false
}

// HasGroup проверяет группу как на плоском уровне claims, так и внутри contour assignment.
func HasGroup(identity *Identity, group string) bool {
	if IsOwner(identity) {
		return true
	}
	needle := strings.TrimSpace(group)
	if needle == "" || identity == nil {
		return false
	}

	for _, item := range identity.Groups {
		if strings.EqualFold(strings.TrimSpace(item), needle) {
			return true
		}
	}
	for _, contour := range identity.Contours {
		for _, item := range contour.Groups {
			if strings.EqualFold(strings.TrimSpace(item), needle) {
				return true
			}
		}
	}

	return false
}

// HasPermission проверяет плоский permission list и вложенные contour permissions.
func HasPermission(identity *Identity, permission string) bool {
	if IsOwner(identity) {
		return true
	}
	needle := strings.TrimSpace(permission)
	if needle == "" || identity == nil {
		return false
	}

	for _, item := range identity.Permissions {
		if strings.EqualFold(strings.TrimSpace(item), needle) {
			return true
		}
	}
	for _, contour := range identity.Contours {
		for _, item := range contour.Permissions {
			if strings.EqualFold(strings.TrimSpace(item), needle) {
				return true
			}
		}
	}

	return false
}

// RequireContour возвращает стандартную forbidden-ошибку при отсутствии доступа к контуру.
func RequireContour(identity *Identity, contour string) error {
	if HasContour(identity, contour) {
		return nil
	}

	return serviceerrors.Forbidden("auth.contour_forbidden", "Недостаточно прав для доступа к контуру")
}

// RequirePermission возвращает стандартную forbidden-ошибку при отсутствии permission.
func RequirePermission(identity *Identity, permission string) error {
	if HasPermission(identity, permission) {
		return nil
	}

	return serviceerrors.Forbidden("auth.permission_forbidden", "Недостаточно прав для выполнения операции")
}

func normalizeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, rawValue := range values {
		value := strings.TrimSpace(rawValue)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}

	return result
}

func normalizeRules(input map[string][]string) map[string][]string {
	if len(input) == 0 {
		return nil
	}

	result := make(map[string][]string, len(input))
	for rawKey, rawValues := range input {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			continue
		}

		values := normalizeStrings(rawValues)
		if len(values) == 0 {
			continue
		}

		result[key] = values
	}
	if len(result) == 0 {
		return nil
	}

	return result
}

func normalizeContours(input []ContourAccess) []ContourAccess {
	if len(input) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(input))
	result := make([]ContourAccess, 0, len(input))
	for _, rawItem := range input {
		contour := strings.TrimSpace(rawItem.Contour)
		if contour == "" {
			continue
		}

		key := strings.ToLower(contour)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		result = append(result, ContourAccess{
			Contour:     contour,
			Role:        strings.TrimSpace(rawItem.Role),
			Groups:      normalizeStrings(rawItem.Groups),
			Permissions: normalizeStrings(rawItem.Permissions),
		})
	}
	if len(result) == 0 {
		return nil
	}

	return result
}
