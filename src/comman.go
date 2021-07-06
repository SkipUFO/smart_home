package main

type ecsCommanMessage struct {
	ID     int32  `json:"id"`
	Type   string `json:"type"`
	Sender string `json:"sender"`
	Folder int32  `json:"folder"`
}

type commanAuditEvent int

const (
	// Received Сообщение получено
	Received commanAuditEvent = iota - 1

	// Read Сообщение прочитано
	Read
	// Unread Сообщение установлено непрочитанным
	Unread

	// CopiedFrom Сообщение скопировано из папки
	CopiedFrom
	// CopiedTo Сообщение скопировано в папку
	CopiedTo

	// Mark Сообщение отмечено флажком
	Mark
	// Unmark Снята отметка флажком у сообщения
	Unmark

	// Archived Сообщение отправлено в архив
	Archived
	// Restored Сообщение восстановлено из архива
	Restored
	// DeniedAutoArchive Сообщению установлен признак запрета автоархивирования
	DeniedAutoArchive

	// SentToReparse Сообщение отправлено на репарсинг
	SentToReparse
	// Reparsed Сообщение обработано заново
	Reparsed

	// QueuedToSend Сообщение поставлено в очередь на отправку
	QueuedToSend
	// Sent Сообщение отправлено
	Sent
	// NotSent Сообщение не отправлено
	NotSent

	// Changed Сообщение изменено
	Changed
	// ChangedDraft Черновик изменен
	ChangedDraft
	// SavedDraft Черновик добавлен
	SavedDraft
)

func (t commanAuditEvent) Code() int {
	return int(t)
}

func (t commanAuditEvent) Name() string {
	switch t {
	case Received:
		return "message-received"
	case Read:
		return "message-read"
	case Unread:
		return "message-set-unread"
	case CopiedFrom:
		return "message-copied-from"
	case CopiedTo:
		return "message-copied-to"
	case Mark:
		return "message-marked"
	case Unmark:
		return "message-unmarked"
	case Archived:
		return "message-archived"
	case Restored:
		return "message-restored"
	case DeniedAutoArchive:
		return "message-set-deny-to-auto-archive"
	case SentToReparse:
		return "message-sent-to-reparse"
	case Reparsed:
		return "message-reparsed"
	case QueuedToSend:
		return "message-queued-to-send"
	case Sent:
		return "message-sent"
	case NotSent:
		return "message-not-sent"
	case Changed:
		return "message-changed"
	case ChangedDraft:
		return "draft-changed"
	case SavedDraft:
		return "message-saved-as-draft"
	}

	return ""
}

func (t commanAuditEvent) String(language int) string {
	lang := language - 1 // т.к. массив с нуля

	if lang > 2 {
		return ""
	}
	switch t {
	case Received:
		return []string{"Message received", "Сообщение получено"}[lang]
	case Read:
		return []string{"Message read", "Сообщение прочитано"}[lang]
	case Unread:
		return []string{"Message set unread", "Сообщение установлено прочитанным"}[lang]
	case CopiedFrom:
		return []string{"Message copied from", "Сообщение скопировано из папки"}[lang]
	case CopiedTo:
		return []string{"Message copied to", "Сообщение скопировано в папку"}[lang]
	case Mark:
		return []string{"Message marked", "Сообщение помечено флажком"}[lang]
	case Unmark:
		return []string{"Message unmarked", "У сообщения снята отметка флажком"}[lang]
	case Archived:
		return []string{"Message archived", "Сообщение отправлено в архив"}[lang]
	case Restored:
		return []string{"Message restored", "Сообщение восстановлено из архива"}[lang]
	case DeniedAutoArchive:
		return []string{"Message set deny to auto archive", "Сообщению установлен запрет на автоархивирование"}[lang]
	case SentToReparse:
		return []string{"Message sent to reparse", "Сообщение отправлено на повторную обработку"}[lang]
	case Reparsed:
		return []string{"Message reparsed", "Сообщение повторно обработано"}[lang]
	case QueuedToSend:
		return []string{"Message queued to send", "Сообщение поставлено в очередь на отправку"}[lang]
	case Sent:
		return []string{"Message sent", "Сообщение отправлено"}[lang]
	case NotSent:
		return []string{"Message not sent", "Сообщение не отправлено"}[lang]
	case Changed:
		return []string{"Message changed", "Сообщение изменено"}[lang]
	case ChangedDraft:
		return []string{"Draft changed", "Черновик изменен"}[lang]
	case SavedDraft:
		return []string{"Message saved as draft", "Сообщение сохранено в черновики"}[lang]
	}

	return ""
}
