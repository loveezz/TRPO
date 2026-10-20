"""Тесты программы «Список слов» (практическая работа 8).

Запуск: python -m unittest discover -s wordlist -v

1. DataTests  — тесты, проверяющие ДАННЫЕ (метод «черного ящика»):
   классы эквивалентности и граничные значения входной фразы.
2. LogicTests — тесты, проверяющие ЛОГИКУ (метод «белого ящика»):
   каждая ветвь split_words / is_separator / reverse_word выполняется
   хотя бы одним тестом (покрытие маршрутов).
3. FormTests  — восходящее (пошаговое) тестирование: уже проверенные
   функции подключаются к форме, проверяется связка кнопка → список →
   надпись с перевернутым словом.
"""

import unittest

from words import is_separator, reverse_word, split_by_spaces, split_words


class DataTests(unittest.TestCase):
    """Черный ящик: ожидаемый результат берется из спецификации, а не из кода."""

    CASES = [
        # (класс данных, входная фраза, ожидаемый список слов)
        ("пустая строка", "", []),
        ("только пробелы", "     ", []),
        ("только разделители", " .,;:!? ", []),
        ("одно слово", "Привет", ["Привет"]),
        ("одна буква", "я", ["я"]),
        ("обычная фраза", "Мама мыла раму", ["Мама", "мыла", "раму"]),
        ("несколько пробелов подряд", "Мама   мыла  раму", ["Мама", "мыла", "раму"]),
        ("пробелы по краям", "  мир  ", ["мир"]),
        ("перевод строки и табуляция", "раз\nдва\tтри", ["раз", "два", "три"]),
        ("точка и запятая", "Раз,два.Три", ["Раз", "два", "Три"]),
        ("знаки препинания", "Да! Нет? Может: быть; (наверное)",
         ["Да", "Нет", "Может", "быть", "наверное"]),
        ("кавычки-ёлочки и тире", "«Слово» — это сила", ["Слово", "это", "сила"]),
        ("многоточие", "Ну…что ж...", ["Ну", "что", "ж"]),
        ("дефис внутри слова", "кто-то пришел", ["кто-то", "пришел"]),
        ("составное название", "Санкт-Петербург и Ту-154", ["Санкт-Петербург", "и", "Ту-154"]),
        ("дефис как тире", "Москва - столица", ["Москва", "столица"]),
        ("числа", "В 2026 году 12,5 градусов", ["В", "2026", "году", "12", "5", "градусов"]),
        ("латиница и кириллица", "Go и Python", ["Go", "и", "Python"]),
        ("буква ё и регистр", "Ёжик ЁЛКА", ["Ёжик", "ЁЛКА"]),
        ("неразрывный пробел", "100 рублей", ["100", "рублей"]),
    ]

    def test_equivalence_classes(self):
        for name, text, expected in self.CASES:
            with self.subTest(name):
                self.assertEqual(split_words(text), expected)

    def test_word_order_preserved(self):
        self.assertEqual(split_words("в, б, а"), ["в", "б", "а"])

    def test_duplicates_kept(self):
        self.assertEqual(split_words("да, да, да"), ["да", "да", "да"])

    def test_long_text(self):
        text = "слово " * 10000
        words = split_words(text)
        self.assertEqual(len(words), 10000)
        self.assertTrue(all(w == "слово" for w in words))

    def test_very_long_word(self):
        word = "а" * 100000
        self.assertEqual(split_words(word), [word])

    def test_invalid_type(self):
        for bad in (None, 123, ["a", "b"], b"bytes"):
            with self.subTest(repr(bad)):
                with self.assertRaises(TypeError):
                    split_words(bad)

    def test_reverse_data(self):
        cases = [("", ""), ("а", "а"), ("да", "ад"), ("кот", "ток"), ("Привет", "тевирП"),
                 ("шалаш", "шалаш"), ("кто-то", "от-отк"), ("2026", "6202"), ("Go", "oG")]
        for word, expected in cases:
            with self.subTest(word):
                self.assertEqual(reverse_word(word), expected)

    def test_reverse_invalid_type(self):
        with self.assertRaises(TypeError):
            reverse_word(None)

    def test_first_version_spaces_only(self):
        """Первая версия (п. 5.2) знаки препинания оставляла в словах —
        именно этот недостаток устранен в п. 5.3."""
        self.assertEqual(split_by_spaces("Раз, два."), ["Раз,", "два."])
        self.assertEqual(split_words("Раз, два."), ["Раз", "два"])


class LogicTests(unittest.TestCase):
    """Белый ящик: тесты построены по тексту функций, по одному на маршрут."""

    # --- is_separator: 5 маршрутов ---
    def test_sep_route_separator_char(self):
        self.assertTrue(is_separator("a,b", 1))          # ch in SEPARATORS

    def test_sep_route_ordinary_char(self):
        self.assertFalse(is_separator("abc", 1))         # не разделитель, не дефис

    def test_sep_route_hyphen_inside_word(self):
        self.assertFalse(is_separator("a-b", 1))         # left_ok и right_ok

    def test_sep_route_hyphen_at_start(self):
        self.assertTrue(is_separator("-ab", 0))          # i == 0 -> left_ok False

    def test_sep_route_hyphen_at_end(self):
        self.assertTrue(is_separator("ab-", 2))          # i+1 == len -> right_ok False

    def test_sep_route_hyphen_between_spaces(self):
        self.assertTrue(is_separator("a - b", 2))

    # --- split_words: ветви цикла и завершения ---
    def test_split_loop_not_entered(self):
        self.assertEqual(split_words(""), [])            # цикл 0 раз, start < 0

    def test_split_separator_without_word(self):
        self.assertEqual(split_words(",,"), [])          # разделитель при start < 0

    def test_split_separator_closes_word(self):
        self.assertEqual(split_words("ab,"), ["ab"])     # разделитель при start >= 0

    def test_split_word_continues(self):
        self.assertEqual(split_words("abc"), ["abc"])    # не-разделитель при start >= 0

    def test_split_last_word_after_loop(self):
        self.assertEqual(split_words(",ab"), ["ab"])     # завершение после цикла

    def test_split_mixed_route(self):
        self.assertEqual(split_words(" a, -b- c"), ["a", "b", "c"])

    # --- reverse_word: цикл 0, 1 и много раз ---
    def test_reverse_loop_zero_iterations(self):
        self.assertEqual(reverse_word(""), "")
        self.assertEqual(reverse_word("x"), "x")

    def test_reverse_loop_one_iteration(self):
        self.assertEqual(reverse_word("ab"), "ba")
        self.assertEqual(reverse_word("abc"), "cba")     # средний символ на месте

    def test_reverse_loop_many_iterations(self):
        self.assertEqual(reverse_word("abcdef"), "fedcba")

    # --- свойства (инварианты) логики ---
    def test_reverse_is_involution(self):
        for w in split_words("Мама мыла раму, а кто-то — нет! 12345"):
            with self.subTest(w):
                self.assertEqual(reverse_word(reverse_word(w)), w)
                self.assertEqual(len(reverse_word(w)), len(w))
                self.assertEqual(reverse_word(w)[0], w[-1])
                self.assertEqual(reverse_word(w)[-1], w[0])

    def test_words_contain_no_separators(self):
        text = "Раз, два; три: (четыре) — «пять»… шесть-семь!"
        for w in split_words(text):
            for i in range(len(w)):
                self.assertFalse(is_separator(w, i), f"разделитель в слове {w!r}")

    def test_words_are_substrings_in_order(self):
        text = "Первое, второе. Третье!"
        pos = 0
        for w in split_words(text):
            found = text.find(w, pos)
            self.assertGreaterEqual(found, 0)
            pos = found + len(w)


class FormTests(unittest.TestCase):
    """Восходящее тестирование: проверенные модули words.py + форма."""

    @classmethod
    def setUpClass(cls):
        try:
            import tkinter as tk
            from app import WordListApp
            cls.root = tk.Tk()
        except Exception as exc:  # нет графической среды
            raise unittest.SkipTest(f"tkinter недоступен: {exc}")
        # Окно прозрачно, но отображено: виртуальное событие <<ListboxSelect>>
        # доходит только до отрисованных виджетов.
        cls.root.attributes("-alpha", 0.0)
        cls.app = WordListApp(cls.root)
        cls.root.update()

    @classmethod
    def tearDownClass(cls):
        cls.root.destroy()

    def test_form_title_and_empty_memo(self):
        self.assertEqual(self.root.title(), "Список слов. Программа Тахмазов И.Ш.")

    def test_memo_cleared_on_start(self):
        import tkinter as tk
        from app import WordListApp
        top = tk.Toplevel(self.root)
        try:
            fresh = WordListApp(top)
            self.assertEqual(fresh.memo.get("1.0", "end-1c"), "")
            self.assertEqual(fresh.listbox.size(), 0)
        finally:
            top.destroy()

    def test_button_fills_list(self):
        app = self.app
        app.memo.delete("1.0", "end")
        app.memo.insert("1.0", "Мама мыла раму, а папа — окно.")
        app.button.invoke()
        self.assertEqual(list(app.listbox.get(0, "end")),
                         ["Мама", "мыла", "раму", "а", "папа", "окно"])
        self.assertEqual(app.count_label.cget("text"), "Слов: 6")

    def test_repeated_click_replaces_list(self):
        app = self.app
        app.memo.delete("1.0", "end")
        app.memo.insert("1.0", "один два")
        app.button.invoke()
        app.button.invoke()
        self.assertEqual(app.listbox.size(), 2)

    def test_select_shows_reversed(self):
        app = self.app
        app.memo.delete("1.0", "end")
        app.memo.insert("1.0", "Привет, мир")
        app.button.invoke()
        app.listbox.selection_clear(0, "end")
        app.listbox.selection_set(0)
        app.listbox.event_generate("<<ListboxSelect>>")
        self.root.update()
        self.assertEqual(app.reversed_var.get(), "Перевернутое слово: тевирП")

    def test_no_selection_clears_label(self):
        app = self.app
        app.listbox.selection_clear(0, "end")
        self.assertEqual(app.show_reversed(), "")
        self.assertEqual(app.reversed_var.get(), "")

    def test_empty_input(self):
        app = self.app
        app.memo.delete("1.0", "end")
        app.button.invoke()
        self.assertEqual(app.listbox.size(), 0)
        self.assertEqual(app.count_label.cget("text"), "Слов: 0")


if __name__ == "__main__":
    unittest.main(verbosity=2)
