# ArgoCD Applications - Документация

## Обзор

Директория `apps/` содержит манифесты ArgoCD Application для автоматического развертывания приложения `ac-integration` в Kubernetes кластер. Реализован GitOps-подход с двумя окружениями: **dev** и **prod**.

---

## Архитектура

```
┌─────────────────────────────────────────────────────────────────────┐
│                         ArgoCD Controller                           │
│                                                                     │
│  ┌─────────────────────────┐     ┌─────────────────────────┐       │
│  │ ac-integration-dev.yaml │     │ ac-integration-prod.yaml│       │
│  │                         │     │                         │       │
│  │  Namespace: argocd      │     │  Namespace: argocd      │       │
│  │  Target: dev            │     │  Target: prod           │       │
│  │  Replicas: 1            │     │  Replicas: 2            │       │
│  │  Auto-sync: enabled     │     │  Auto-sync: enabled     │       │
│  └───────────┬─────────────┘     └───────────┬─────────────┘       │
│              │                               │                      │
└──────────────┼───────────────────────────────┼──────────────────────┘
               │                               │
               ▼                               ▼
┌──────────────────────────┐     ┌──────────────────────────┐
│    GitOps Repository     │     │    GitOps Repository     │
│  overlays/dev/           │     │  overlays/prod/          │
│  kustomization.yaml      │     │  kustomization.yaml      │
└──────────────────────────┘     └──────────────────────────┘
               │                               │
               ▼                               ▼
┌──────────────────────────┐     ┌──────────────────────────┐
│   Kubernetes Cluster     │     │   Kubernetes Cluster     │
│   Namespace: dev         │     │   Namespace: prod        │
│   - Deployment           │     │   - Deployment           │
│   - Service              │     │   - Service              │
└──────────────────────────┘     └──────────────────────────┘
```

---

## Файлы

### 1. ac-integration-dev.yaml

**Назначение:** ArgoCD Application для непрерывного развертывания в окружение **dev**.

#### Конфигурация

| Параметр | Значение | Описание |
|----------|----------|----------|
| `metadata.name` | `ac-integration-dev` | Имя приложения в ArgoCD |
| `metadata.namespace` | `argocd` | Namespace где создается Application |
| `spec.project` | `default` | Проект ArgoCD |
| `spec.source.repoURL` | `git@github.com:uzgidro/GitOps-Repo.git` | URL GitOps репозитория |
| `spec.source.targetRevision` | `main` | Ветка для отслеживания |
| `spec.source.path` | `apps/ac-integration/overlays/dev` | Путь к Kustomize overlay |
| `spec.destination.server` | `https://kubernetes.default.svc` | Целевой кластер |
| `spec.destination.namespace` | `dev` | Целевой namespace |

#### Политика синхронизации

```yaml
syncPolicy:
  automated:
    prune: true      # Удаляет ресурсы, отсутствующие в Git
    selfHeal: true   # Автоматически восстанавливает при drift
  syncOptions:
    - CreateNamespace=true  # Создает namespace если отсутствует
    - PruneLast=true        # Удаление после применения новых ресурсов
```

---

### 2. ac-integration-prod.yaml

**Назначение:** ArgoCD Application для непрерывного развертывания в окружение **prod**.

#### Конфигурация

| Параметр | Значение | Описание |
|----------|----------|----------|
| `metadata.name` | `ac-integration-prod` | Имя приложения в ArgoCD |
| `metadata.namespace` | `argocd` | Namespace где создается Application |
| `spec.project` | `default` | Проект ArgoCD |
| `spec.source.repoURL` | `git@github.com:uzgidro/GitOps-Repo.git` | URL GitOps репозитория |
| `spec.source.targetRevision` | `main` | Ветка для отслеживания |
| `spec.source.path` | `apps/ac-integration/overlays/prod` | Путь к Kustomize overlay |
| `spec.destination.server` | `https://kubernetes.default.svc` | Целевой кластер |
| `spec.destination.namespace` | `prod` | Целевой namespace |

---

## Сравнение окружений

| Аспект | Dev | Prod |
|--------|-----|------|
| **Application Name** | `ac-integration-dev` | `ac-integration-prod` |
| **Target Namespace** | `dev` | `prod` |
| **Overlay Path** | `overlays/dev` | `overlays/prod` |
| **Replicas** | 1 | 2 |
| **Memory Request** | 128Mi | 256Mi |
| **Memory Limit** | 256Mi | 512Mi |
| **CPU Request** | 100m | 200m |
| **CPU Limit** | 500m | 1000m |
| **Обновление** | Автоматическое (push в main) | Ручное (promote workflow) |

---

## Установка

### Предварительные требования

1. **ArgoCD** установлен в кластере
2. **SSH-ключ** настроен для доступа к GitOps репозиторию
3. **Namespace `argocd`** существует

### Развертывание Applications

```bash
# Применить оба Application манифеста
kubectl apply -f apps/

# Или по отдельности
kubectl apply -f apps/ac-integration-dev.yaml
kubectl apply -f apps/ac-integration-prod.yaml
```

### Проверка статуса

```bash
# Через kubectl
kubectl get applications -n argocd

# Через ArgoCD CLI
argocd app list
argocd app get ac-integration-dev
argocd app get ac-integration-prod
```

---

## Управление

### Ручная синхронизация

```bash
# ArgoCD CLI
argocd app sync ac-integration-dev
argocd app sync ac-integration-prod
```

### Откат (Rollback)

```bash
argocd app rollback ac-integration-dev <REVISION_NUMBER>
argocd app rollback ac-integration-prod <REVISION_NUMBER>

# Просмотр истории
argocd app history ac-integration-dev
argocd app history ac-integration-prod
```

---

## Ссылки

- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [ArgoCD Application CRD](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/)
- [Kustomize](https://kustomize.io/)
