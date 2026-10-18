package prayertime

// IshaRule — как определяется время Иши.
type IshaRule struct {
	Type  string  // "angle" — по углу солнца под горизонтом; "minutesAfterMaghrib" — N минут после Магриба
	Value float64 // градусы или минуты
}

// Method — метод расчёта времени намазов.
type Method struct {
	ID        string            // совпадает с CalculationMethod в iOS
	Title     map[string]string // язык → название
	FajrAngle float64           // угол солнца под горизонтом для Фаджра, градусы
	Isha      IshaRule
	AladhanID int // номер метода в api.aladhan.com (параметр method)
}

func angle(v float64) IshaRule   { return IshaRule{Type: "angle", Value: v} }
func minutes(v float64) IshaRule { return IshaRule{Type: "minutesAfterMaghrib", Value: v} }

// Methods — все методы Aladhan. Первые 6 — исходные из ТЗ (§6), порядок не менять.
// Идентификаторы СТРОГО совпадают с iOS.
var Methods = []Method{
	{ID: "muslimWorldLeague", AladhanID: 3, FajrAngle: 18, Isha: angle(17), Title: map[string]string{
		"en": "Muslim World League", "ru": "Всемирная исламская лига", "ar": "رابطة العالم الإسلامي", "tg": "Робитаи ҷаҳони ислом"}},
	{ID: "northAmerica", AladhanID: 2, FajrAngle: 15, Isha: angle(15), Title: map[string]string{
		"en": "ISNA (North America)", "ru": "ISNA (Северная Америка)", "tg": "ISNA (Амрикои Шимолӣ)"}},
	{ID: "egypt", AladhanID: 5, FajrAngle: 19.5, Isha: angle(17.5), Title: map[string]string{
		"en": "Egyptian General Authority", "ru": "Египетское управление геодезии", "tg": "Идораи генералии геодезии Миср"}},
	{ID: "ummAlQura", AladhanID: 4, FajrAngle: 18.5, Isha: minutes(90), Title: map[string]string{
		"en": "Umm al-Qura, Makkah", "ru": "Умм аль-Кура, Мекка", "ar": "أم القرى", "tg": "Умм-ул-Қуро, Макка"}},
	{ID: "karachi", AladhanID: 1, FajrAngle: 18, Isha: angle(18), Title: map[string]string{
		"en": "University of Karachi", "ru": "Университет Карачи", "tg": "Донишгоҳи Карачӣ"}},
	{ID: "turkey", AladhanID: 13, FajrAngle: 18, Isha: angle(17), Title: map[string]string{
		"en": "Diyanet, Türkiye", "ru": "Диянет, Турция", "tg": "Диёнат, Туркия"}},

	// Остальные методы Aladhan
	{ID: "moonsightingCommittee", AladhanID: 15, FajrAngle: 18, Isha: angle(18), Title: map[string]string{
		"en": "Moonsighting Committee Worldwide", "ru": "Moonsighting Committee", "tg": "Кумитаи рӯъяти ҳилол"}},
	{ID: "russia", AladhanID: 14, FajrAngle: 16, Isha: angle(15), Title: map[string]string{
		"en": "Spiritual Administration of Muslims of Russia", "ru": "ДУМ России", "tg": "Идораи рӯҳонии мусулмонони Русия"}},
	{ID: "gulf", AladhanID: 8, FajrAngle: 19.5, Isha: minutes(90), Title: map[string]string{
		"en": "Gulf Region", "ru": "Страны Персидского залива", "tg": "Минтақаи Халиҷи Форс"}},
	{ID: "kuwait", AladhanID: 9, FajrAngle: 18, Isha: angle(17.5), Title: map[string]string{
		"en": "Kuwait", "ru": "Кувейт", "tg": "Кувайт"}},
	{ID: "qatar", AladhanID: 10, FajrAngle: 18, Isha: minutes(90), Title: map[string]string{
		"en": "Qatar", "ru": "Катар", "tg": "Қатар"}},
	{ID: "dubai", AladhanID: 16, FajrAngle: 18.2, Isha: angle(18.2), Title: map[string]string{
		"en": "Dubai, UAE", "ru": "Дубай, ОАЭ", "tg": "Дубай, АМА"}},
	{ID: "jordan", AladhanID: 23, FajrAngle: 18, Isha: angle(18), Title: map[string]string{
		"en": "Ministry of Awqaf, Jordan", "ru": "Министерство вакфов, Иордания", "tg": "Вазорати авқоф, Урдун"}},
	{ID: "tehran", AladhanID: 7, FajrAngle: 17.7, Isha: angle(14), Title: map[string]string{
		"en": "Institute of Geophysics, Tehran", "ru": "Институт геофизики, Тегеран", "tg": "Институти геофизикаи Теҳрон"}},
	{ID: "jafari", AladhanID: 0, FajrAngle: 16, Isha: angle(14), Title: map[string]string{
		"en": "Shia Ithna-Ashari (Jafari), Qum", "ru": "Джафари (Кум)", "tg": "Ҷаъфарӣ (Қум)"}},
	{ID: "jakim", AladhanID: 17, FajrAngle: 20, Isha: angle(18), Title: map[string]string{
		"en": "JAKIM, Malaysia", "ru": "JAKIM, Малайзия", "tg": "JAKIM, Малайзия"}},
	{ID: "singapore", AladhanID: 11, FajrAngle: 20, Isha: angle(18), Title: map[string]string{
		"en": "MUIS, Singapore", "ru": "MUIS, Сингапур", "tg": "MUIS, Сингапур"}},
	{ID: "kemenag", AladhanID: 20, FajrAngle: 20, Isha: angle(18), Title: map[string]string{
		"en": "KEMENAG, Indonesia", "ru": "KEMENAG, Индонезия", "tg": "KEMENAG, Индонезия"}},
	{ID: "france", AladhanID: 12, FajrAngle: 12, Isha: angle(12), Title: map[string]string{
		"en": "UOIF, France", "ru": "UOIF, Франция", "tg": "UOIF, Фаронса"}},
	{ID: "portugal", AladhanID: 22, FajrAngle: 18, Isha: minutes(77), Title: map[string]string{
		"en": "Islamic Community of Lisbon", "ru": "Исламская община Лиссабона", "tg": "Ҷамоати исломии Лиссабон"}},
	{ID: "tunisia", AladhanID: 18, FajrAngle: 18, Isha: angle(18), Title: map[string]string{
		"en": "Tunisia", "ru": "Тунис", "tg": "Тунис"}},
	{ID: "algeria", AladhanID: 19, FajrAngle: 18, Isha: angle(17), Title: map[string]string{
		"en": "Algeria", "ru": "Алжир", "tg": "Алҷазоир"}},
	{ID: "morocco", AladhanID: 21, FajrAngle: 19, Isha: angle(17), Title: map[string]string{
		"en": "Morocco", "ru": "Марокко", "tg": "Марокаш"}},
}

// MethodByID ищет метод по идентификатору.
func MethodByID(id string) (Method, bool) {
	for _, m := range Methods {
		if m.ID == id {
			return m, true
		}
	}
	return Method{}, false
}
