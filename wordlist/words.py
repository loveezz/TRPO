"""Логика программы «Список слов» (практическая работа 8).

Модуль не зависит от интерфейса, поэтому тестируется отдельно
от формы (тестирование модулей, восходящий подход).
"""

# Разделители слов (п. 5.3): пробельные символы и знаки препинания.
# Дефис разделителем не считается, если стоит внутри слова («кто-то»).
SEPARATORS = frozenset(
    " \t\r\n\v\f "
    ".,;:!?…"
    "()[]{}<>"
    "\"'«»„“”‘’`"
    "/\\|*+=_~^#№%&@$"
    "—–"
)
HYPHEN = "-"


def split_by_spaces(text):
    """Первая версия (п. 5.2): слова разделяются только пробелами."""
    words = []
    current = ""
    for ch in text:
        if ch == " ":
            if current:
                words.append(current)
                current = ""
        else:
            current += ch
    if current:
        words.append(current)
    return words


def is_separator(text, i):
    """Является ли символ text[i] разделителем слов.

    Дефис — часть слова, только если слева и справа от него буквы
    или цифры («Санкт-Петербург», «Ту-154»); иначе это тире/разделитель.
    """
    ch = text[i]
    if ch in SEPARATORS:
        return True
    if ch == HYPHEN:
        left_ok = i > 0 and text[i - 1].isalnum()
        right_ok = i + 1 < len(text) and text[i + 1].isalnum()
        return not (left_ok and right_ok)
    return False


def split_words(text):
    """Формирует список слов фразы с учетом всех разделителей (п. 5.3).

    Алгоритм: посимвольный просмотр строки; символ-не-разделитель
    добавляется к текущему слову, разделитель завершает текущее слово,
    если оно не пустое. После цикла завершается последнее слово.
    """
    if text is None:
        raise TypeError("ожидалась строка, получено None")
    if not isinstance(text, str):
        raise TypeError(f"ожидалась строка, получено {type(text).__name__}")
    words = []
    start = -1  # индекс начала текущего слова, -1 — слова нет
    for i in range(len(text)):
        if is_separator(text, i):
            if start >= 0:
                words.append(text[start:i])
                start = -1
        elif start < 0:
            start = i
    if start >= 0:
        words.append(text[start:])
    return words


def reverse_word(word):
    """Переворот слова (п. 5.4): последняя буква становится первой,
    первая — последней. Обмен симметричных символов от краев к середине."""
    if not isinstance(word, str):
        raise TypeError(f"ожидалась строка, получено {type(word).__name__}")
    chars = list(word)
    i, j = 0, len(chars) - 1
    while i < j:
        chars[i], chars[j] = chars[j], chars[i]
        i += 1
        j -= 1
    return "".join(chars)
