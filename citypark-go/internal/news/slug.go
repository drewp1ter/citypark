package news

import (
	"strings"
	"unicode"
)

const maxSlugLength = 80

var translit = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e", 'ж': "zh", 'з': "z", 'и': "i",
	'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t",
	'у': "u", 'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "shch", 'ъ': "", 'ы': "y", 'ь': "",
	'э': "e", 'ю': "yu", 'я': "ya",
}

// Slugify transliterates Russian to Latin (а→a, б→b, в→v, г→g, д→d, е→e, ё→e, ж→zh, з→z, и→i, й→y, к→k, л→l, м→m,
// н→n, о→o, п→p, р→r, с→s, т→t, у→u, ф→f, х→kh, ц→ts, ч→ch, ш→sh, щ→shch, ъ→"", ы→y, ь→"", э→e, ю→yu, я→ya),
// lowercases, turns every run of other characters into one "-", trims "-" and cuts to 80 characters
// (without a trailing "-"). Returns "news" when nothing is left.
func Slugify(title string) string {
	var b strings.Builder
	pendingDash := false
	for _, r := range title {
		r = unicode.ToLower(r)
		part, ok := translit[r]
		switch {
		case ok:
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			part = string(r)
		default:
			pendingDash = true
			continue
		}
		if part == "" {
			continue
		}
		if pendingDash && b.Len() > 0 {
			b.WriteByte('-')
		}
		pendingDash = false
		b.WriteString(part)
	}
	slug := b.String()
	if len(slug) > maxSlugLength {
		slug = slug[:maxSlugLength]
	}
	slug = strings.TrimRight(slug, "-")
	if slug == "" {
		return "news"
	}
	return slug
}
