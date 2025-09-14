;;; $DOOMDIR/config.el -*- lexical-binding: t; -*-

;; karthinks.com/software/emacs-window-management-almanac/
;; notes.justin.vc/config

;; Reconfigure packages with `after!' block wrap, otherwise Doom's defaults may override your settings. E.g.
;;   (after! PACKAGE
;;     (setq x y))
;;
;; Exceptions to this:
;;   - Setting file/directory variables (like `org-directory')
;;   - Setting variables which explicitly tell you to set them before their
;;     package is loaded (see 'C-h v VARIABLE' to look up their documentation).
;;   - Setting doom variables (which start with 'doom-' or '+').
;;
;; Additional Doom configuration functions/macros:
;; - `load!' for loading external *.el files relative to this one
;; - `use-package!' for configuring packages
;; - `after!' for running code after a package has loaded
;; - `add-load-path!rectories to the `load-path', relative to
;;   this file. Emacs searches the `load-path' when you load packages with
;;   `require' or `use-package'.
;; - `map!' for binding new keys
;;
;; See documentation about these functions/macros, by pressing
;; 'K' over the highlighted symbol ('C-c c k' for non-evil users).
;; open documentation for it, including demos of how they are used.
;; Alternatively, use `C-h o' to look up symbols (functions, variables, faces,
;; etc).
;;
;; See implementations with 'gd' over symbol (or 'C-c c d').

;; https://tecosaur.github.io/emacs-config

;; https://git.sr.ht/~morgansmith/sway-ts-mode
;; ;; (load! "./extra/sway-ts-mode")
;; (add-load-path "./extra")
;; (setq treesit-extra-load-path "./extra")

;; (minimap-mode)
(display-time-mode 1)
(global-undo-tree-mode 1)
;; (keycast-tab-line-mode)
;; (+global-word-wrap-mode +1)
(set-language-environment "UTF-8")
(set-default-coding-systems 'utf-8)

(setq nyan-mode t
      nyan-wavy-trail t
      nyan-animate-nyancat t
      doom-modeline-hud t
      doom-theme 'doom-gruvbox
      which-key-idle-delay 0.25
      evil-split-window-below t
      doom-modeline-time-icon t
      evil-vsplit-window-right t
      doom-modeline-persp-name t
      display-time-day-and-date t
      treemacs-git-mode 'extended
      doom-lantern-padded-modeline t
      doom-modeline-major-mode-icon t
      org-directory "~/Documents/org/"
      user-full-name "Mumtahin Farabi"
      display-line-numbers-type 'relative
      which-key-allow-multiple-replacements t ;; Remove 'evil-' in too many popups
      user-mail-address "mfarabi619@gmail.com"
      plstore-cache-passphrase-for-symmetric-encryption t
      browse-url-browser-function 'browse-url-default-browser
      fancy-splash-image "~/MFarabi619/assets/apollyon-emacs.png"
      projectile-project-search-path '("~/workspace/" "~/Documents/")
      doom-font (font-spec :family "JetBrainsMono Nerd Font" :size 20)
      doom-big-font (font-spec :family "JetBrainsMono Nerd Font" :size 40)
      doom-variable-pitch-font (font-spec :family "JetBrainsMono Nerd Font" :size 20))

(after! treemacs
  (setq
   treemacs-position 'left
   treemacs-indent-guide-mode t
   lsp-treemacs-theme "Idea"
   ;; "Eclipse"
   ;; "NetBeans"
   treemacs-git-commit-diff-mode t
   ;; treemacs-load-theme "doom-colors"
   lsp-treemacs-symbols-position-params '((side . left) (slot . 1) (window-width . 35))))

(after! dired
  (setq dirvish-peek-mode t
        dirvish-side-auto-close t
        dirvish-side-follow-mode t
        dirvish-side-display-alist '((side . right) (slot . -1))))

(after! dirvish
  (setq dirvish-default-layout '(1 0.11 0.70)
        dirvish-quick-access-entries
        `(("h" "~/"                          "Home")
          ("t" "~/.local/share/Trash/files/" "Trash")
          ("p" "~/Pictures/"                 "Pictures")
          ("w" "~/workspace/"                "Workspace")
          ("d" "~/Downloads/"                "Downloads")
          ("m" "/mnt/"                       "Mounted drives")
          ("e" ,user-emacs-directory         "Emacs user directory"))))

(use-package! gptel
  :config
  (setq gptel-model 'gpt-4o
        ;; gptel-api-key "your key"
        gptel-default-mode #'org-mode
        gptel-backend (gptel-make-gh-copilot "Copilot")
        gptel-directives '((default     . "You are a large language model living in Doom Emacs and a helpful assistant. Respond concisely. I'm using Doom Emacs with Evil Mode inside Arch Linux with Hyprland. I browse the web with Vivaldi. I also use Nix for configuration management, and write code in Rust.")
                           (programming . "You are a large language model and a careful programmer. Provide code and only code as output without any additional text, prompt or note.")
                           (writing     . "You are a large language model and a writing assistant. Respond concisely.")
                           (chat        . "You are a large language model and a conversation partner. Respond concisely.")))
  (add-hook 'gptel-post-stream-hook 'gptel-auto-scroll
            'gptel-post-response-functions 'gptel-end-of-response))

(use-package! jira
  :config
  (setq jira-api-version 3 ;; Version 2 is also allowed
        ;; jira-tempo-token "foobar123123") ;; https://apidocs.tempo.io
        jira-username "mfarabi619@gmail.com"
        jira-base-url ""
        ;; https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/
        jira-token ""))

(map! :n "C-'" #'+vterm/toggle
      :n "C-l" nil :n "C-l" #'+lazygit/toggle
      :leader :desc "Open Dirvish" "e" #'dirvish
      :leader :desc "Open AI Chat buffer" "d" #'gptel
      :leader :desc "Toggle vterm" "j" #'+vterm/toggle
      :leader :desc "Open Dirvish Side" "[" #'dirvish-side)

(map! :map evil-window-map
      "SPC"       #'rotate-layout
      "<up>"      #'evil-window-up
      "<left>"    #'evil-window-left
      "<down>"    #'evil-window-down
      "<right>"   #'evil-window-right
      "C-<up>"    #'+evil/window-move-up
      "C-<left>"  #'+evil/window-move-left
      "C-<down>"  #'+evil/window-move-down
      "C-<right>" #'+evil/window-move-right)

(set-popup-rule! "*Copilot*"
  :ttl 0
  :size 0.5
  :vslot -4
  :quit nil
  :select t
  :side 'left)

(set-popup-rule! "^\\*Flycheck errors\\*$"
  :size 0.4
  :select t
  :side 'bottom)

(set-popup-rule! "*doom:vterm-popup:*"
  :quit t
  :slot 0
  :ttl nil
  :vslot 0
  :select t
  :width 0.5
  :height 0.5
  :modeline t
  :side 'right)

(set-popup-rule! "*doom:vterm-popup:lazygit*"
  :quit t
  :slot 0
  :vslot 0
  :ttl nil
  :select t
  :width 0.5
  :height 0.5
  :modeline t
  :side 'right)

(defun +lazygit/toggle ()
  "Bring up or reuse a vterm popup running lazygit."
  (interactive)
  (+vterm--configure-project-root-and-display
   nil
   (lambda ()
     (let* ((buffer-name
             (format "*doom:vterm-popup:lazygit-%s*"
                     (if (bound-and-true-p persp-mode)
                         (safe-persp-name (get-current-persp))
                       "main")))
            (buffer (or (cl-loop for buf in (doom-buffers-in-mode 'vterm-mode)
                                 if (equal (buffer-local-value '+vterm--id buf)
                                           buffer-name)
                                 return buf)
                        (get-buffer-create buffer-name)))
            (proc   (get-buffer-process buffer))
            (need-launch (not (and proc (process-live-p proc)))))
       (if-let ((win (get-buffer-window buffer-name)))
           (delete-window win)
         (with-current-buffer buffer
           (unless (eq major-mode 'vterm-mode)
             (vterm-mode))
           (setq-local +vterm--id buffer-name))
         (pop-to-buffer buffer)
         (when need-launch
           (vterm-send-string "lazygit status -sm normal; exit")
           (vterm-send-return)))
       buffer))))

;; (map! :leader ;; Remap switching to last buffer from 'SPC+`' to 'SPC+e'
;;       :desc "Switch to last buffer"
;;       "e" #'evil-switch-to-windows-last-buffer)
;; "e" #'my/switch-to-last-buffer-in-split)

(defun my/switch-to-last-buffer-in-split ()
  "Show last buffer on split screen."
  (interactive)
  (let ((current-buffer (current-buffer)))
    (if (one-window-p)
        (progn
          (split-window-right)
          (evil-switch-to-windows-last-buffer)
          (switch-to-buffer current-buffer)))))

(defadvice! prompt-for-buffer (&rest _)
  :after '(evil-window-split evil-window-vsplit)
  (consult-buffer))

(add-hook
 'pdf-view-mode-hook
 'pdf-view-midnight-minor-mode
 'doom-modeline-mode-hook #'nyan-mode)

(after! evil
  (setq evil-ex-substitute-global t ;; implicit /g flag on evil ex substitution
        evil-escape-key-sequence "jk")
  (add-hook 'evil-local-mode-hook 'turn-on-undo-tree-mode)
  (define-key evil-normal-state-map (kbd "j") 'evil-next-visual-line)
  (define-key evil-normal-state-map (kbd "k") 'evil-previous-visual-line))

(after! org
  (define-key org-mode-map (kbd "C-c C-r") verb-command-map)
  (setq
   org-modern-table-vertical 1
   org-modern-table-horizontal 0.2
   org-link-search-must-match-exact-headline nil
   ;; org-modern-priority-faces
   org-priority-faces '((?A :foreground "#e45649")
                        (?B :foreground "#da8548")
                        (?C :foreground "#0098dd"))
   org-modern-star ["◉" "○" "✸" "✿" "✤" "✜" "◆" "▶"]
   org-modern-list '((43 . "➤") (45 . "–") (42 . "•"))
   ;; org-modern-todo-faces
   org-todo-keyword-faces '(("DONE" :foreground "#50a14f" :weight normal :underline t)
                            ("TODO" :foreground "#7c7c75" :weight normal :underline t)
                            ("BLOCKED" :foreground "#ff9800" :weight normal :underline t)
                            ("CANCELLED" :foreground "#ff6480" :weight normal :underline t)
                            ("INPROGRESS" :foreground "#0098dd" :weight normal :underline t))
   org-todo-keywords  '((sequence "TODO(t)" "INPROGRESS(i)" "BLOCKED(b)" "|" "DONE(d)" "CANCELLED(c)")))
  )

(after! nerd-icons
  (setq nerd-icons-completion-mode t))

(after! pdf-tools
  (setq pdf-view-continuous t))

(after! nov-xwidget
  :demand t
  :after nov
  :config
  (define-key nov-mode-map (kbd "o") 'nov-xwidget-view)
  (add-hook 'nov-mode-hook 'nov-xwidget-inject-all-files))

(after! centaur-tabs-mode
  (setq centaur-tabs-show-count t
        centaur-tabs-gray-out-icons t
        centaur-tabs-enable-key-bindings t
        centaur-tabs-show-navigation-buttons t))

(after! fretboard
  (setq fretboard-fret-count 15)
  (add-hook 'fretboard-mode-hook #'evil-emacs-state))

(after! lsp-mode
  (setq lsp-eslint-run "onSave"
        lsp-format-buffer-on-save t
        ;; lsp-typescript-format-enable t
        lsp-eslint-package-manager "pnpm"
        lsp-typescript-references-code-lens-enabled t
        lsp-typescript-suggest-complete-function-calls t)
  )


(after! which-key
  (pushnew!
   which-key-replacement-alist
   '(("" . "\\`+?evil[-:]?\\(?:a-\\)?\\(.*\\)") . (nil . "◂\\1"))
   '(("\\`g s" . "\\`evilem--?motion-\\(.*\\)") . (nil . "◃\\1"))))

;; You do not need to run 'doom sync' after modifying this file!

(defun my-custom-dashboard-text ()
  "Insert custom text into the Doom dashboard."
  (insert "\"Do not proceed with a mess; messes just grow with time.\" ― Bjarne Stroustrup\n\n"))

;; Find `doom-dashboard-widget-banner` in the list and insert after it
(let ((pos (cl-position #'doom-dashboard-widget-banner +doom-dashboard-functions)))
  (when pos
    (setq +doom-dashboard-functions
          (append (cl-subseq +doom-dashboard-functions 0 (1+ pos))
                  (list #'my-custom-dashboard-text)
                  (cl-subseq +doom-dashboard-functions (1+ pos))))))

;; Doom exposes five (optional) variables for controlling fonts in Doom:
;; - `doom-font' -- primary font to use
;; - `doom-big-font' -- used for `doom-big-font-mode'
;; - `doom-serif-font' -- for `fixed-pitch-serif' face
;; - `doom-symbol-font' -- for symbols
;; - `doom-variable-pitch-font' -- a non-monospace font (where applicable)
;;
;; See 'C-h v doom-font' for documentation and more examples of what they
;; accept. For example:
;;
;; (setq doom-font (font-spec :family "Fira Code" :size 12 :weight 'semi-light))
;; doom-variable-pitch-font (font-spec :family "Fira Sans" :size 13))
;;
;; If you or Emacs can't find your font, use 'M-x describe-font' to look them
;; up, `M-x eval-region' to execute elisp code, and 'M-x doom/reload-font' to
;; refresh your font settings. If Emacs still can't find your font, it likely
;; wasn't installed correctly. Font issues are rarely Doom issues!

;; There are two ways to load a theme. Both assume the theme is installed and
;; available. You can either set `doom-theme' or manually load a theme with the
;; `load-theme' function.

;; https://www.ovistoica.com/blog/2024-7-05-modern-emacs-typescript-web-tsx-config
