package render

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

var ldquo = "\u201c" // "
var rdquo = "\u201d" // "

// renderComponent renders a templ Component to a string.
func renderComponent(c templ.Component) string {
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		return ""
	}
	return buf.String()
}

func Dashboard(d DashboardData) string     { return renderComponent(dashboard(d)) }
func About() string                        { return renderComponent(about()) }
func Lesson(d LessonData) string           { return renderComponent(lesson(d)) }
func Record(d RecordData) string           { return renderComponent(record(d)) }
func Ref(d RefData) string                 { return renderComponent(ref(d)) }
func QuizLibrary(d QuizLibraryData) string { return renderComponent(quizLibrary(d)) }
func Quiz(d QuizData) string               { return renderComponent(quiz(d)) }
func QuizAttempt(d AttemptData) string {
	out := renderComponent(quizAttempt(d))

	// Inject JSON data + JS as raw script tags (templ treats <script>
	// content as raw text, so we append after the templ component).
	questionsJSON, _ := json.Marshal(d.Questions)
	answeredIDs := make([]int64, 0, len(d.AnsweredIDs))
	for id := range d.AnsweredIDs {
		answeredIDs = append(answeredIDs, id)
	}
	answeredResults := map[string]bool{}
	for id, correct := range d.AnsweredResults {
		answeredResults[strconv.FormatInt(id, 10)] = correct
	}
	dataJSON, _ := json.Marshal(map[string]any{
		"attemptId":        d.AttemptID,
		"quizSlug":         d.QuizSlug,
		"quizTitle":        d.QuizTitle,
		"workspace":        d.Workspace.Name,
		"questions":        json.RawMessage(questionsJSON),
		"answeredIds":      answeredIDs,
		"answeredResults":  answeredResults,
		"autoSubmitChoice": d.AutoSubmitChoice,
	})
	out += `<script type="application/json" id="attempt-data">` + string(dataJSON) + `</script>`
	out += `<script>` + quizAttemptJS + `</script>`
	return out
}

func answerSummary(item ReviewItem) string {
	if item.Mode == "choice" {
		idx, _ := strconv.Atoi(item.UserResponse)
		if idx >= 0 && idx < len(item.Options) {
			if item.IsCorrect {
				return item.Options[idx]
			}
			if item.CorrectIndex >= 0 && item.CorrectIndex < len(item.Options) {
				return item.Options[item.CorrectIndex]
			}
		}
	} else {
		if item.IsCorrect {
			return "Known"
		}
		return "Review"
	}
	return ""
}

func choiceReviewDetail(item ReviewItem) string {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	userResp, _ := strconv.Atoi(item.UserResponse)
	var b strings.Builder
	for i, opt := range item.Options {
		if i == item.CorrectIndex {
			b.WriteString(`<div class="flex items-center gap-3 p-3 rounded-lg border-2 border-emerald-600 bg-emerald-100 text-sm text-slate-800 font-medium">`)
			b.WriteString(`<span class="w-6 h-6 rounded-full bg-emerald-600 text-white flex items-center justify-center text-xs shrink-0">`)
			b.WriteByte(letters[i])
			b.WriteString(`</span><span class="font-medium text-slate-900">Correct:</span> `)
			b.WriteString(esc(opt))
			b.WriteString(`<svg class="ml-auto shrink-0 text-emerald-600" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"/></svg></div>`)
		} else if i == userResp {
			b.WriteString(`<div class="flex items-center gap-3 p-3 rounded-lg border-2 border-red-600 bg-red-100 text-sm text-slate-800 font-medium">`)
			b.WriteString(`<span class="w-6 h-6 rounded-full bg-red-600 text-white flex items-center justify-center text-xs shrink-0">`)
			b.WriteByte(letters[i])
			b.WriteString(`</span><span class="font-medium text-slate-900">You answered:</span> `)
			b.WriteString(esc(opt))
			b.WriteString(`<svg class="ml-auto shrink-0 text-red-600" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></div>`)
		} else {
			b.WriteString(`<div class="flex items-center gap-3 p-3 rounded-lg border border-slate-200 bg-white text-sm text-slate-400">`)
			b.WriteString(`<span class="w-6 h-6 rounded-full border border-slate-300 flex items-center justify-center text-xs text-slate-400 shrink-0">`)
			b.WriteByte(letters[i])
			b.WriteString(`</span>`)
			b.WriteString(esc(opt))
			b.WriteString(`</div>`)
		}
	}
	return b.String()
}

func recallReviewDetail(item ReviewItem) string {
	var b strings.Builder
	b.WriteString(`<div class="p-4 rounded-lg border border-dashed border-slate-300 bg-slate-50 text-sm text-slate-700 leading-relaxed">`)
	b.WriteString(esc(item.RevealText))
	b.WriteString(`</div>`)
	if item.IsCorrect {
		b.WriteString(`<div class="text-xs text-emerald-600">You marked this as known.</div>`)
	} else {
		b.WriteString(`<div class="text-xs text-red-600">You marked this for review.</div>`)
	}
	return b.String()
}

func QuizReview(d QuizReviewData) string {
	return renderComponent(quizReview(d))
}
func NotFound(title, message string) string { return renderComponent(notFound(title, message)) }

// workspaceMeta formats the lesson/record/ref/quiz counts for a dashboard card.
func workspaceMeta(w WorkspaceCard) string {
	counts := make([]string, 0, 4)
	if w.LessonCount > 0 {
		counts = append(counts, fmt.Sprintf("%d lessons", w.LessonCount))
	}
	if w.RecordCount > 0 {
		counts = append(counts, fmt.Sprintf("%d records", w.RecordCount))
	}
	if w.RefCount > 0 {
		counts = append(counts, fmt.Sprintf("%d refs", w.RefCount))
	}
	if w.QuizCount > 0 {
		counts = append(counts, fmt.Sprintf("%d quizzes", w.QuizCount))
	}
	meta := strings.Join(counts, " · ")
	if meta == "" {
		meta = "empty"
	}
	return meta
}

// bigIcon returns an svg with its size attributes set to px×px, used for
// empty-state illustrations where the default 16px is too small.
func bigIcon(svg string, px int) string {
	size := strconv.Itoa(px)
	s := strings.Replace(svg, `width="16" height="16"`, `width="`+size+`" height="`+size+`"`, 1)
	return strings.Replace(s, `width="20" height="20"`, `width="`+size+`" height="`+size+`"`, 1)
}

// orEmpty returns s if non-empty, otherwise fallback.
func orEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// BestScoreText returns the display text for a quiz entry's score badge.
func (q QuizEntry) BestScoreText() string {
	if q.BestScore < 0 {
		return "not started"
	}
	return fmt.Sprintf("%d/%d", q.BestScore, q.BestTotal)
}

// quizBadgeClass returns the Tailwind classes for a quiz entry's score badge.
func quizBadgeClass(q QuizEntry) string {
	if q.BestScore < 0 {
		return "bg-slate-100 text-slate-400"
	}
	if q.BestScore == q.BestTotal {
		return "bg-emerald-100 text-emerald-600"
	}
	return "bg-amber-100 text-amber-600"
}

func statusTag(status string) string {
	if status == "superseded" {
		return `<span class="inline-flex items-center bg-red-100 text-red-600 text-xs font-medium px-2 py-0.5 rounded">superseded</span>`
	}
	return `<span class="inline-flex items-center bg-emerald-100 text-emerald-600 text-xs font-medium px-2 py-0.5 rounded">active</span>`
}
