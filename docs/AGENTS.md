# Agentes de IA

Assets compartilhados de IA do projeto: skills, templates e specs versionados no repositório.

Todo desenvolvedor que clonar o repositório usa os mesmos prompts, skills e templates, sem configuração local adicional.

---

## Estrutura

```
.ai/
├── agents.yaml                 — convenções, paths e regras do projeto
├── scripts/
│   └── sync_skills.sh          — copia .ai/skills → .agents/skills
├── skills/                     — fonte canônica das skills compartilhadas
│   ├── feature-architect/
│   ├── feature-checker/
│   └── feature-scaffold/
├── templates/                  — templates base para novas features
│   ├── FEATURE_SPEC.md
│   └── IMPLEMENTATION_TRACKER.md
└── specs/{task-slug}/          — especificações geradas por feature
    ├── FEATURE_SPEC.md
    └── IMPLEMENTATION_TRACKER.md

.agents/
└── skills/                     — runtime path consumido pelo agente (espelho de .ai/skills)
```

---

## Setup

Após clonar o repositório, sincronize as skills para o runtime:

```bash
./.ai/scripts/sync_skills.sh
```

Isso copia `.ai/skills` → `.agents/skills`, que é o diretório lido pelo GitHub Copilot.

---

## Skills disponíveis

| Skill | Descrição |
|-------|-----------|
| `feature-architect` | Gera spec técnica completa + tracker de implementação |
| `feature-checker` | Verifica progresso real do tracker por fase e step |
| `feature-scaffold` | Cria estrutura inicial de spec a partir de template |

Para invocar uma skill no Copilot, referencie o arquivo `SKILL.md` correspondente em `.ai/skills/{skill-name}/SKILL.md`.

---

## Templates

| Template | Uso |
|----------|-----|
| `.ai/templates/FEATURE_SPEC.md` | Documento técnico completo da feature |
| `.ai/templates/IMPLEMENTATION_TRACKER.md` | Acompanhamento por fase, step e verificação |

---

## Convenções

- Fonte de verdade das skills: `.ai/skills`.
- `.agents/skills` é espelho de runtime — não editar diretamente.
- Ao alterar qualquer skill em `.ai/skills`, execute o script de sync e faça commit das duas pastas.
- Specs geradas ficam em `.ai/specs/{task-slug}/` e são versionadas junto ao código.
