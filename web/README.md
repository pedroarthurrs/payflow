# i9pay Web

Frontend SPA do i9pay, construído com Vue 3 + TypeScript + Vite.

## Tech Stack

- **Vue 3** (Composition API + `<script setup>`)
- **TypeScript 5.6**
- **Vite 6** (dev server + build)
- **Pinia** (state management)
- **Vue Router 4** (SPA routing com guards)
- **Axios** (HTTP client com interceptors)
- **Vitest + Vue Test Utils** (testes unitários)
- **MSW** (mock de API nos testes)

## Estrutura

```
web/
├── src/
│   ├── main.ts                  # Bootstrap: Vue + Pinia + Router
│   ├── App.vue                  # Root component (layout condicional por rota)
│   │
│   ├── components/              # Componentes reutilizáveis
│   │   ├── AccessDenied.vue     # Página 403
│   │   ├── AppLayout.vue        # Layout autenticado (sidebar/header)
│   │   ├── ConfirmDialog.vue    # Modal de confirmação genérico
│   │   ├── DataTable.vue        # Tabela com ações (edit/delete)
│   │   ├── FormField.vue        # Campo de formulário com validação
│   │   ├── LoadingSpinner.vue   # Indicador de carregamento
│   │   └── ToastContainer.vue   # Container de notificações
│   │
│   ├── composables/             # Lógica reutilizável (hooks)
│   │   ├── useAuth.ts           # Login, logout, verificação de role
│   │   ├── useCrud.ts           # Operações CRUD genéricas via HTTP
│   │   ├── useErrorHandler.ts   # Tratamento padronizado de erros HTTP
│   │   └── useToast.ts          # Sistema de notificações (singleton)
│   │
│   ├── router/                  # Configuração de rotas
│   │   └── index.ts             # Rotas, guards (auth, RBAC, guest)
│   │
│   ├── services/                # Camada HTTP
│   │   └── http.ts              # Axios instance, interceptors, auto-refresh
│   │
│   ├── stores/                  # Estado global (Pinia)
│   │   └── auth.ts              # Tokens JWT, user info, login/logout
│   │
│   ├── types/                   # Tipagens TypeScript
│   │   └── index.ts             # Interfaces: Customer, Address, User, API errors
│   │
│   └── views/                   # Páginas (uma por rota)
│       ├── DashboardView.vue
│       ├── LoginView.vue
│       ├── addresses/
│       │   └── AddressFormView.vue
│       ├── customers/
│       │   ├── CustomerListView.vue
│       │   ├── CustomerDetailView.vue
│       │   └── CustomerFormView.vue
│       └── users/
│           ├── UserListView.vue
│           └── UserFormView.vue
│
├── index.html                   # Entry point HTML
├── env.d.ts                     # Declarações de tipo (Vite client, .vue modules)
├── vite.config.ts               # Vite: dev server, proxy, alias, vitest
├── tsconfig.json                # Project references root
├── tsconfig.app.json            # Config TS para código da aplicação
├── tsconfig.node.json           # Config TS para vite.config.ts
└── package.json                 # Dependências e scripts
```

## Scripts

```bash
pnpm dev          # Dev server na porta 3000
pnpm build        # Type-check + build de produção
pnpm preview      # Preview do build de produção
pnpm test         # Roda todos os testes (single run)
pnpm test:watch   # Testes em modo watch
```

## Padrões

- **Testes colocados** junto ao código fonte (`*.test.ts` ao lado do `*.vue` ou `*.ts`)
- **Composables** encapsulam lógica reutilizável, views consomem via composição
- **useCrud** é genérico — recebe o `basePath` e retorna `list`, `getById`, `create`, `update`, `remove`
- **Auth flow**: JWT com auto-refresh via interceptor do Axios; tokens no `localStorage`
- **RBAC no frontend**: guard no router verifica role hierarchy (`owner > admin > member`)
- **Proxy**: `/api` é redirecionado para o backend Go (porta 8080) via Vite proxy

## Variáveis de Ambiente

- **`VITE_API_PROXY_TARGET`**: alvo do proxy do Vite no ambiente de desenvolvimento. Exemplo: `http://localhost:8080` ou `http://api:8080` em Docker.
- **`VITE_API_BASE_URL`**: base consumida pelo browser fora do proxy do Vite. Exemplo: `http://localhost:8080`.
