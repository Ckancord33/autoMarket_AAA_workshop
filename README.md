# AutoMarket - Workshop

AutoMarket es una plataforma web para la comercialización de vehículos usados desarrollada en **Go** sobre el framework **Gin**. El sistema opera como un **servicio de renderizado en servidor (Server-Side Rendering - SSR)** con plantillas HTML dinámicas, integrando un esquema integral de seguridad basado en el modelo **AAA (Authentication, Authorization, Auditing)** construido exclusivamente con las librerías nativas del lenguaje. Toda la experiencia de usuario, autenticación de sesiones basada en cookies seguras y administración de publicaciones se procesa directamente en el servidor sin dependencias de frameworks cliente externos.

---

## Objetivo de Seguridad: Modelo AAA con Librerías Nativas

El núcleo del taller consiste en diseñar e integrar un ecosistema de seguridad autónomo y desacoplado de suites de terceros (como librerías externas de JWT u OAuth):

* **Authentication (Autenticación):** Emisión y validación de tokens de sesión firmados criptográficamente mediante hashes simétricos **HMAC-SHA256**. El mecanismo codifica el payload en base64 URL-safe, verifica la integridad, valida la expiración temporal de forma determinista y persiste la sesión de navegación en cookies HTTP seguras (`httpOnly`).
* **Authorization (Autorización - RBAC):** Control de acceso basado en roles (**Admin**, **Seller**) ejecutado mediante middlewares interceptores en Gin. Cada solicitud a vistas protegidas (`/seller`, `/admin`) es evaluada contra los permisos requeridos antes de despachar el controlador y renderizar la plantilla HTML.
* **Auditing (Auditoría):** Captura y persistencia de trazas operativas no bloqueantes (`sync.RWMutex` / estructuras en memoria y persistencia JSON en `data/audit_log.json`). Se audita quién ejecuta la acción, el recurso accedido, el método HTTP, la dirección IP de origen, el código de estado retornado y la latencia de respuesta en milisegundos.

---

## Modelo Relacional y Entidades

El dominio de datos unifica las especificaciones mecánicas del vehículo, las fotografías del automotor (`images`) y las condiciones comerciales de la oferta en una sola entidad de publicación (`PUBLICATION`), vinculando al vendedor (`USER`) y los registros de trazabilidad de seguridad (`AUDIT_LOG`).

```mermaid
---
config:
  theme: redux-color
  look: classic
title: AutoMarket Relational Diagram (Fused Vehicle & Listing)
---
erDiagram
    direction TB

    USER {
        int idUser PK
        string email UK
        string password
        string name
        string role
        string phone
        string city
        datetime createdAt
        boolean isActive
    }

    PUBLICATION {
        int idPublication PK
        int idUser FK
        string brand
        string model
        int year
        string color
        string condition
        decimal price
        string description
        string status
        boolean isAuthorized
        datetime publishedAt 
        datetime soldAt "nullable"
        int mileage
        array images
    }

    AUDIT_LOG {
        bigint idLog PK
        int idUser FK "nullable"
        string role
        string action
        string method
        string endpoint
        string ipAddress
        int statusCode
        int latencyMs
        string details "nullable"
        datetime timestamp
    }

    USER ||--o{ PUBLICATION : "publishes"
    USER |o--o{ AUDIT_LOG : "generates"
```

### Descripción de Entidades

* **`USER`**: Almacena las credenciales de acceso (`email`, `password`), información del perfil (`name`, `phone`, `city`), fecha de creación (`createdAt`), estado activo (`isActive`) y rol asignado (`role`: Admin, Seller) para el control de acceso basado en roles (RBAC).
* **`PUBLICATION`**: Modelo consolidado con clave primaria `idPublication` que unifica las propiedades físicas del vehículo (`brand`, `model`, `year`, `color`, `condition`, `mileage`), la lista de imágenes subidas (`images`) y las condiciones del mercado (`price`, `description`, `status`, `isAuthorized`, `publishedAt`, `soldAt`), vinculado al vendedor (`idUser`).
* **`AUDIT_LOG`**: Registro inmutable de trazabilidad y eventos de seguridad del modelo AAA. Captura al usuario ejecutor (`idUser` nullable para eventos anónimos o no autenticados), rol (`role`), acción ejecutada (`action`, `method`, `endpoint`), dirección IP (`ipAddress`), telemetría de respuesta (`statusCode`, `latencyMs`), metadatos contextuales (`details`) y marca de tiempo (`timestamp`).

#### Relaciones del Modelo

* **`USER ||--o{ PUBLICATION`**: Un usuario puede crear y publicar múltiples ofertas de vehículos.
* **`USER |o--o{ AUDIT_LOG`**: Un usuario genera registros de auditoría al interactuar con el sistema (relación opcional en el extremo del usuario para registrar también acciones no autenticadas).

---

## Arquitectura del Proyecto: Servicio Web SSR

El sistema está estructurado bajo una **arquitectura por capas (*Layered Architecture*)** desacoplada y opera como un **servicio de renderizado en el servidor (Server-Side Rendering - SSR)**:

1. **Capa de Presentación Web (SSR con Gin):** El motor web compila las plantillas ubicadas en `web/templates/` mediante `router.LoadHTMLGlob()`. Los controladores responden inyectando modelos directamente en las vistas HTML (`c.HTML()`), evitando dependencias de navegadores pesados o frameworks de JavaScript en el cliente.
2. **Navegación e Interacción:** El flujo de usuario (autenticación, registro, publicación con carga de archivos binarios en `web/static/uploads`, reporte de venta y moderación) se basa en formularios estándar HTML5 (`POST`) y enlaces de navegación semántica (`GET`).
3. **Gestión de Sesión Criptográfica:** El token de autenticación (JWT nativo HMAC-SHA256) emitido al iniciar sesión se almacena de forma persistente en una cookie HTTP (`Set-Cookie: token`), permitiendo que el middleware intercepte y valide la identidad del usuario en cada solicitud de página web de manera transparente.
4. **Capa de Servicios y Dominio:** Los controladores delegan la lógica a `services/`, quienes aplican las reglas de negocio y consumen la capa de persistencia `repositories/`.
5. **Persistencia Concurrente:** El almacenamiento opera sobre archivos JSON en el directorio `data/` usando exclusión mutua (`sync.RWMutex`) para garantizar consistencia y lecturas O(1) en memoria.

```plaintext
autoMarket_AAA_workshop/
├── cmd/
│   └── server/
│       └── main.go                  # Punto de entrada. Inicializa Gin, middlewares y plantillas.
├── data/                            # Persistencia local en archivos JSON (Runtime)
│   ├── audit_log.json               # Trazas de auditoría operacional AAA
│   ├── publication.json             # Catálogo de vehículos y ofertas
│   └── user.json                    # Cuentas, credenciales y roles de usuario
├── internal/
│   ├── models/                      # Entidades de dominio
│   │   ├── audit_log.go             # Entidad AuditLog para trazabilidad AAA
│   │   ├── publication.go           # Entidad Publication y especificaciones del vehículo
│   │   └── user.go                  # Entidad User y roles (Admin, Seller)
│   │
│   ├── handlers/                    # Controladores web (Transporte y Renderizado SSR)
│   │   ├── publication_handler.go   # Vistas de catálogo, detalle, vendedor y admin
│   │   └── user_handler.go          # Vistas de login, registro, logout y administración
│   │
│   ├── middleware/                  # Interceptores del modelo AAA
│   │   ├── audit.go                 # Trazabilidad operativa asíncrona no bloqueante
│   │   └── auth.go                  # Validación criptográfica JWT y control RBAC
│   │
│   ├── services/                    # Casos de uso y reglas de negocio puras
│   │   ├── publication_service.go   # Flujo de publicación, moderación, venta y consulta
│   │   └── user_service.go          # Lógica de registro, hash de claves, login y permisos
│   │
│   └── repositories/                # Capa de persistencia (Operaciones sobre archivos JSON)
│       ├── audit_repo.go            # Almacenamiento seguro e indexado de logs de auditoría
│       ├── publication_repo.go      # Lectura y escritura concurrente de publicaciones
│       └── user_repo.go             # Lectura y escritura concurrente de usuarios
│
├── pkg/                             # Módulos transversales desacoplados del dominio
│   ├── response/
│   │   └── http_response.go         # Formato estándar de respuestas y errores de seguridad
│   └── security/                    # Utilidades de seguridad criptográfica nativa (AAA)
│       ├── jwt.go                   # Generación y validación HMAC-SHA256 de tokens
│       └── password.go              # Hash seguro con salt aleatorio (SHA-256)
│
├── web/                             # Capa de presentación (SSR con Gin)
│   ├── static/                      # Archivos estáticos servidos en /static
│   │   ├── css/                     # Estilos visuales de la aplicación (main.css)
│   │   └── uploads/                 # Fotografías subidas de los automotores
│   └── templates/                   # Plantillas HTML procesadas por el motor web
│       ├── admin.html               # Panel de moderación y gestión para el Administrador
│       ├── index.html               # Catálogo público de vehículos
│       ├── login.html               # Formulario de inicio de sesión
│       ├── publication_detail.html  # Ficha técnica del vehículo y contacto del vendedor
│       ├── register.html            # Formulario de registro de vendedores
│       └── seller.html              # Panel de publicación y gestión para el Vendedor
│
├── config/
│   └── config.go                    # Constantes y configuración interna del proyecto
├── go.mod
└── go.sum
```

---

## Puesta en Marcha (Cómo correr el proyecto)

### Prerrequisitos

* **Go**: Versión 1.22 o superior instalada.
* **Persistencia sin base de datos externa**: Por requerimientos del taller no se utilizan dependencias externas más allá del framework **Gin**. El almacenamiento y persistencia de datos se gestiona directamente en archivos JSON dentro del directorio `data/` (`user.json`, `publication.json`, `audit_log.json`) empleando las librerías estándar de Go (`encoding/json`, `os`, `sync`).

### Pasos de Ejecución

1. **Clonar el repositorio:**
   ```bash
   git clone <URL_DEL_REPOSITORIO>
   cd autoMarket_AAA_workshop
   ```

2. **Descargar dependencias:**
   ```bash
   go mod download
   ```

3. **Ejecutar el servidor:**
   ```bash
   go run cmd/server/main.go
   ```

4. **Acceso a la plataforma web:**
   Abre tu navegador en `http://localhost:8080`.

---

## Declaración de Uso Responsable de Inteligencia Artificial

En concordancia con los lineamientos de integridad académica y transparencia técnica establecidos por el **Institute of Electrical and Electronics Engineers (IEEE)** y la **Association for Computing Machinery (ACM)**, se declara el uso de herramientas de Inteligencia Artificial Generativa basadas en Modelos de Lenguaje Grande (LLMs) durante el ciclo de vida del proyecto.

Dichas herramientas se emplearon como asistencia técnica para:
* La formulación, contraste y evaluación comparativa de decisiones de diseño de arquitectura de software y modelado relacional.
* La asistencia en la sintaxis, estructuración de handlers, diseño de middlewares AAA y optimización de fragmentos de código fuente en Go.
* El apoyo en la redacción, estructuración y estandarización técnica de la presente documentación.

> **Compromiso de auditoría:** Los autores ratifican que los sistemas de IA carecen de estatus de autoría. Todo el código fuente, esquemas relacionales y decisiones de seguridad fueron revisados, probados, auditados e integrados manualmente por el equipo de desarrollo, asumiendo la total responsabilidad por el funcionamiento, calidad y seguridad de la solución.

---

## Contribuyentes y Créditos

* **Equipo de Desarrollo del Taller:** Estudiantes de la asignatura de Ciberseguridad / Ingeniería de Sistemas.

---

## Licencia

Este proyecto está distribuido bajo la licencia MIT. Para conocer los términos y condiciones de uso, copia, modificación y distribución, consulte el archivo [LICENSE](LICENSE) ubicado en la raíz del repositorio.