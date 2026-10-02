package discovery

import (
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

// KindForContainer classifies a loot source by the block or entity holding it.
func KindForContainer(container string) domain.SourceKind {
	if strings.Contains(container, "suspicious_") {
		return domain.KindArchaeology
	}
	return domain.KindContainer
}
