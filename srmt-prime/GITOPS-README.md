# GitOps Repository - ac-integration

## Обзор

Данный репозиторий реализует **GitOps-подход** для развертывания приложения `ac-integration` в Kubernetes. Используется **Kustomize** для управления конфигурациями и **ArgoCD** для непрерывного развертывания.

---

## Структура репозитория

```
ac-integration/
├── README.md                          # Краткое описание
├── GITOPS-README.md                   # Эта документация
├── .gitignore                         # Git ignore правила
│
├── apps/                              # ArgoCD Application манифесты
│   ├── README.md                      # Документация ArgoCD Apps
│   ├── ac-integration-dev.yaml        # Application для dev
│   └── ac-integration-prod.yaml       # Application для prod
│
├── base/                              # Базовые Kubernetes манифесты
│   ├── kustomization.yaml             # Kustomize конфигурация
│   ├── deployment.yaml                # Deployment ac-integration
│   └── service.yaml                   # Service ac-integration (ClusterIP)
│
└── overlays/                          # Окружение-специфичные оверлеи
    ├── dev/                           # Dev окружение
    │   ├── kustomization.yaml         # Кастомизация для dev
    │   ├── sealed-config.yaml         # SealedSecret для dev
    │   └── service-nodeport.yaml      # Патч: Service NodePort для dev
    └── prod/                          # Prod окружение
        ├── kustomization.yaml         # Кастомизация для prod
        ├── sealed-config.yaml         # SealedSecret для prod
        └── ingress.yaml               # Ingress для внешнего доступа
```

---

## Сравнение окружений

| Аспект | Dev | Prod |
|--------|-----|------|
| **Namespace** | `dev` | `prod` |
| **Replicas** | 1 | 2 |
| **Memory Request** | 128Mi | 256Mi |
| **Memory Limit** | 256Mi | 512Mi |
| **CPU Request** | 100m | 200m |
| **CPU Limit** | 500m | 1000m |
| **Network Access** | NodePort (30481) | Ingress (ac-integration.speedwagon.uz) |
| **Update Trigger** | Auto (push) | Manual (promote) |

---

## Механизм обновления образов

### Dev Environment

1. **Push в main/master** триггерит `build.yaml`
2. CI/CD собирает образ и пушит в Docker Hub
3. CI/CD обновляет `overlays/dev/kustomization.yaml`
4. ArgoCD детектирует изменение и синхронизирует

### Prod Environment

1. **Ручной запуск** `promote.yaml` с указанием `image_tag`
2. Проверка существования образа в реестре
3. Одобрение через GitHub Environment
4. CI/CD обновляет `overlays/prod/kustomization.yaml`
5. ArgoCD синхронизирует production

---

## Работа с Kustomize

### Просмотр результирующих манифестов

```bash
# Dev окружение
kubectl kustomize overlays/dev/

# Prod окружение
kubectl kustomize overlays/prod/
```

### Прямое применение (без ArgoCD)

```bash
# Dev
kubectl apply -k overlays/dev/

# Prod
kubectl apply -k overlays/prod/
```

---

## Sealed Secrets

Для хранения конфиденциальной конфигурации используются **Sealed Secrets**.

### Создание Sealed Secret

```bash
# Создание манифеста секрета
kubectl create secret generic ac-integration-config \
  --from-file=config.yaml=config/dev.yaml \
  --dry-run=client -o yaml > secret-plain.yaml

# Запечатывание секрета
kubeseal --cert key.pem \
  < secret-plain.yaml > overlays/dev/sealed-config.yaml

# Удаление plain secret
rm secret-plain.yaml
```

---

## Ссылки

- [Kustomize Documentation](https://kustomize.io/)
- [Kubernetes Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)
- [ArgoCD](https://argo-cd.readthedocs.io/)
