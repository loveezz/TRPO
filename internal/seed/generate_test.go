package seed

import (
	"math/rand"
	"testing"

	"hrselect/internal/models"
	"hrselect/internal/service"
)

// Сгенерированные кандидаты должны проходить ту же валидацию, что и
// введенные через форму, и иметь 2–8 разных навыков.
func TestGenCandidateIsValid(t *testing.T) {
	skills := make([]models.Skill, len(genSkillNames))
	for i, name := range genSkillNames {
		skills[i] = models.Skill{ID: uint(i + 1), Name: name}
	}
	rnd := rand.New(rand.NewSource(1))
	for n := 1; n <= 1000; n++ {
		c := genCandidate(rnd, n, skills)
		if err := service.ValidateCandidate(c); err != nil {
			t.Fatalf("кандидат %d (%+v) не проходит валидацию: %v", n, c, err)
		}
		if len(c.Skills) < 2 || len(c.Skills) > 8 {
			t.Fatalf("кандидат %d: %d навыков, ожидалось 2..8", n, len(c.Skills))
		}
		seen := map[uint]bool{}
		for _, s := range c.Skills {
			if seen[s.ID] {
				t.Fatalf("кандидат %d: навык %q повторяется", n, s.Name)
			}
			seen[s.ID] = true
		}
	}
	if len(genSkillWeights) != len(genSkillNames) {
		t.Fatalf("число весов (%d) не совпадает с числом навыков (%d)", len(genSkillWeights), len(genSkillNames))
	}
}
