package seed

import (
	"fmt"
	"log"
	"math/rand"

	"hrselect/internal/models"

	"gorm.io/gorm"
)

// Набор значений для генерации демонстрационных кандидатов
// (проверка производительности подбора на большом объеме данных).
var (
	genMaleFirst    = []string{"Александр", "Дмитрий", "Максим", "Иван", "Артем", "Никита", "Михаил", "Егор", "Андрей", "Кирилл", "Илья", "Роман"}
	genMaleLast     = []string{"Иванов", "Смирнов", "Кузнецов", "Попов", "Васильев", "Петров", "Соколов", "Михайлов", "Новиков", "Федоров", "Морозов", "Волков"}
	genFemaleFirst  = []string{"Анна", "Мария", "Елена", "Ольга", "Наталья", "Екатерина", "Дарья", "Полина", "Софья", "Виктория", "Алина", "Ксения"}
	genFemaleLast   = []string{"Иванова", "Смирнова", "Кузнецова", "Попова", "Васильева", "Петрова", "Соколова", "Михайлова", "Новикова", "Федорова", "Морозова", "Волкова"}
	genCities       = []string{"Екатеринбург", "Екатеринбург", "Москва", "Москва", "Казань", "Пермь", "Новосибирск", "Челябинск", "Санкт-Петербург"}
	genPositions    = []string{"Python Developer", "Backend Developer", "Junior Python Developer", "Senior Backend Developer", "Go Developer", "DevOps Engineer", "Fullstack Developer", "Data Engineer"}
	genEducation    = []string{"Высшее", "Высшее", "Высшее", "Среднее профессиональное", "Неоконченное высшее"}
	genEmployment   = []string{"Полная занятость", "Полная занятость", "Удаленно", "Частичная занятость"}
	genSkillNames   = []string{"Python", "FastAPI", "Django", "PostgreSQL", "SQL", "Docker", "Git", "Linux", "Redis", "REST API", "Go", "Kafka", "Kubernetes", "HTML/CSS", "JavaScript"}
	genSkillWeights = []int{10, 5, 4, 6, 6, 6, 9, 5, 3, 5, 3, 2, 2, 3, 3} // частота навыка у кандидатов
)

// GenerateCandidates дополняет таблицу кандидатов сгенерированными записями
// до общего количества total. Повторный запуск ничего не добавляет, если
// кандидатов уже не меньше total. Генерация воспроизводима (фиксированный seed).
func GenerateCandidates(db *gorm.DB, total int) error {
	if total <= 0 {
		return nil
	}
	var count int64
	if err := db.Model(&models.Candidate{}).Count(&count).Error; err != nil {
		return err
	}
	need := total - int(count)
	if need <= 0 {
		return nil
	}

	skills := make([]models.Skill, len(genSkillNames))
	for i, name := range genSkillNames {
		if err := db.Where("LOWER(name) = LOWER(?)", name).FirstOrCreate(&skills[i], models.Skill{Name: name}).Error; err != nil {
			return err
		}
	}

	rnd := rand.New(rand.NewSource(int64(count) + 2026))
	items := make([]models.Candidate, need)
	for i := range items {
		items[i] = genCandidate(rnd, int(count)+i+1, skills)
	}
	if err := db.CreateInBatches(items, 200).Error; err != nil {
		return err
	}
	log.Printf("[INFO] seed: сгенерировано кандидатов: %d (всего %d)", need, total)
	return nil
}

func genCandidate(rnd *rand.Rand, n int, skills []models.Skill) models.Candidate {
	gender, first, last := "Мужской", pick(rnd, genMaleFirst), pick(rnd, genMaleLast)
	if rnd.Intn(2) == 0 {
		gender, first, last = "Женский", pick(rnd, genFemaleFirst), pick(rnd, genFemaleLast)
	}
	exp := float64(rnd.Intn(25)) / 2 // 0 … 12 лет с шагом полгода
	salary := 60000 + int(exp*15000) + rnd.Intn(16)*5000
	return models.Candidate{
		FullName:          first + " " + last,
		Age:               20 + int(exp) + rnd.Intn(8),
		Gender:            gender,
		Email:             fmt.Sprintf("candidate%04d@example.com", n),
		Phone:             fmt.Sprintf("+7 9%02d %03d-%02d-%02d", rnd.Intn(100), rnd.Intn(1000), rnd.Intn(100), rnd.Intn(100)),
		City:              pick(rnd, genCities),
		Position:          pick(rnd, genPositions),
		ExperienceYears:   exp,
		Education:         pick(rnd, genEducation),
		EmploymentType:    pick(rnd, genEmployment),
		SalaryExpectation: salary,
		About:             "Сгенерированный кандидат для проверки производительности подбора.",
		Skills:            genSkills(rnd, skills),
	}
}

// genSkills выбирает от 2 до 8 разных навыков с учетом их частоты.
func genSkills(rnd *rand.Rand, skills []models.Skill) []models.Skill {
	sum := 0
	for _, w := range genSkillWeights {
		sum += w
	}
	k := 2 + rnd.Intn(7)
	taken := make(map[int]bool, k)
	out := make([]models.Skill, 0, k)
	for len(out) < k {
		r := rnd.Intn(sum)
		i := 0
		for r >= genSkillWeights[i] {
			r -= genSkillWeights[i]
			i++
		}
		if !taken[i] {
			taken[i] = true
			out = append(out, skills[i])
		}
	}
	return out
}

func pick(rnd *rand.Rand, items []string) string {
	return items[rnd.Intn(len(items))]
}
