"""Программа «Список слов» (практическая работа 8).

Форма: поле ввода фразы (аналог Memo), кнопка переноса слов в список
(Button), список слов (ListBox) и надпись под списком с перевернутым
написанием выбранного слова. Для всех компонентов заданы всплывающие
подсказки. Запуск: python wordlist/app.py
"""

import tkinter as tk
from tkinter import ttk

try:
    from words import split_words, reverse_word
except ImportError:  # запуск как пакета: python -m wordlist.app
    from .words import split_words, reverse_word

AUTHOR = "Тахмазов И.Ш."
TITLE = f"Список слов. Программа {AUTHOR}"


class ToolTip:
    """Всплывающая подсказка: в tkinter нет свойства Hint/ShowHint,
    поэтому подсказка реализована вспомогательным классом."""

    DELAY_MS = 500

    def __init__(self, widget, text):
        self.widget = widget
        self.text = text
        self.tip = None
        self.after_id = None
        widget.bind("<Enter>", self._schedule, add="+")
        widget.bind("<Leave>", self._hide, add="+")
        widget.bind("<ButtonPress>", self._hide, add="+")

    def _schedule(self, _event=None):
        self._cancel()
        self.after_id = self.widget.after(self.DELAY_MS, self.show)

    def _cancel(self):
        if self.after_id is not None:
            self.widget.after_cancel(self.after_id)
            self.after_id = None

    def show(self):
        if self.tip is not None:
            return
        x = self.widget.winfo_rootx() + 16
        y = self.widget.winfo_rooty() + self.widget.winfo_height() + 4
        self.tip = tk.Toplevel(self.widget)
        self.tip.wm_overrideredirect(True)
        self.tip.wm_geometry(f"+{x}+{y}")
        tk.Label(self.tip, text=self.text, background="#ffffe1", relief="solid",
                 borderwidth=1, padx=6, pady=3, justify="left").pack()

    def _hide(self, _event=None):
        self._cancel()
        if self.tip is not None:
            self.tip.destroy()
            self.tip = None


class WordListApp:
    def __init__(self, root):
        self.root = root
        root.title(TITLE)
        root.geometry("520x600")
        root.minsize(420, 460)

        frame = ttk.Frame(root, padding=12)
        frame.pack(fill="both", expand=True)

        ttk.Label(frame, text="Фраза:").pack(anchor="w")
        self.memo = tk.Text(frame, height=6, wrap="word", font=("Segoe UI", 11))
        self.memo.pack(fill="x")
        self.memo.delete("1.0", "end")  # поле Memo очищается при запуске
        ToolTip(self.memo, "Введите фразу. Слова разделяются пробелами\n"
                           "и знаками препинания: . , ; : ! ? ( ) « » —")

        self.button = ttk.Button(frame, text="Сформировать список слов", command=self.fill_list)
        self.button.pack(fill="x", pady=8)
        ToolTip(self.button, "Перенести слова фразы в список (Ctrl+Enter)")
        root.bind("<Control-Return>", lambda _e: self.fill_list())

        self.reversed_var = tk.StringVar(value="")
        self.reversed_label = ttk.Label(frame, textvariable=self.reversed_var,
                                        font=("Segoe UI", 12, "bold"))
        self.reversed_label.pack(side="bottom", anchor="w", pady=(4, 0))
        ToolTip(self.reversed_label, "Перевернутое написание выбранного слова")

        self.count_label = ttk.Label(frame, text="Слов: 0")
        self.count_label.pack(side="bottom", anchor="w", pady=(6, 0))

        ttk.Label(frame, text="Список слов:").pack(anchor="w")
        list_frame = ttk.Frame(frame)
        list_frame.pack(fill="both", expand=True)
        self.listbox = tk.Listbox(list_frame, height=8, font=("Segoe UI", 11), activestyle="dotbox",
                                  exportselection=False)
        scroll = ttk.Scrollbar(list_frame, orient="vertical", command=self.listbox.yview)
        self.listbox.configure(yscrollcommand=scroll.set)
        self.listbox.pack(side="left", fill="both", expand=True)
        scroll.pack(side="right", fill="y")
        self.listbox.bind("<<ListboxSelect>>", self.show_reversed)
        ToolTip(self.listbox, "Выберите слово, чтобы увидеть его\nперевернутое написание")


    def fill_list(self):
        """Обработчик кнопки: разбить фразу и заполнить список."""
        words = split_words(self.memo.get("1.0", "end-1c"))
        self.listbox.delete(0, "end")
        for w in words:
            self.listbox.insert("end", w)
        self.count_label.config(text=f"Слов: {len(words)}")
        self.reversed_var.set("")
        return words

    def show_reversed(self, _event=None):
        """Обработчик выбора слова: вывести перевернутое слово под списком."""
        sel = self.listbox.curselection()
        if not sel:
            self.reversed_var.set("")
            return ""
        word = self.listbox.get(sel[0])
        rev = reverse_word(word)
        self.reversed_var.set(f"Перевернутое слово: {rev}")
        return rev


def main():
    root = tk.Tk()
    WordListApp(root)
    root.mainloop()


if __name__ == "__main__":
    main()
