# AutoMarket — Endpoints del Backend y Rutas del Frontend

Guía clara y resumida de todos los endpoints de la API REST (Backend) y las rutas de la interfaz web Server-Side Rendering (Frontend).

---

## 1. Backend (API REST - Formato JSON)

Los endpoints del backend se comunican mediante JSON y utilizan autenticación por Bearer Token (JWT / HMAC nativo) o sesión en cabeceras.

### 👤 Usuarios y Autenticación (`/users`)

| Método | Endpoint | Acceso / Rol | Descripción |
| :--- | :--- | :--- | :--- |
| `POST` | `/users/` | Público | **Registro de usuario:** Crea una cuenta nueva (por defecto rol `seller`) con nombre, correo, teléfono, ciudad y contraseña. Retorna datos de usuario y token. |
| `POST` | `/users/login` | Público | **Inicio de sesión:** Valida credenciales (`email` y `password`) y retorna el token de autenticación junto con los datos del usuario. |
| `DELETE` | `/users/:id` | Protegido (`admin`) | **Eliminar usuario:** Elimina al usuario indicado por ID y borra en cascada todas sus publicaciones asociadas. |

---

### 🚗 Publicaciones de Vehículos (`/publications`)

| Método | Endpoint | Acceso / Rol | Descripción |
| :--- | :--- | :--- | :--- |
| `GET` | `/publications/` | Público | **Listar publicaciones:** Obtiene las publicaciones disponibles. Admite filtros por query params (marca, modelo, año, precio mínimo/máximo, estado). |
| `GET` | `/publications/:id` | Público | **Detalle de publicación:** Retorna en formato JSON la ficha técnica completa del vehículo (se activa si la petición lleva cabecera `Accept: application/json`). |
| `POST` | `/publications/` | Protegido (`seller`, `admin`) | **Crear publicación:** Registra un nuevo vehículo con sus características mecánicas y fotos. Se guarda inicialmente en estado `pendiente` y sin autorización (`isAuthorized: false`). |
| `PATCH` | `/publications/:id` | Protegido (`seller`, `admin`) | **Actualizar publicación:** Modifica campos de una publicación existente (como precio, descripción o estado). |
| `POST` | `/publications/:id/authorize` | Protegido (`admin`) | **Aprobar publicación:** Autoriza el vehículo (`isAuthorized: true`) y cambia su estado a `publicada` para que aparezca en el catálogo público. |

---

## 2. Frontend (Rutas Web SSR - HTML)

Estas rutas renderizan páginas HTML del lado del servidor (SSR) mediante el motor de plantillas de Gin y gestionan la sesión a través de cookies HTTP seguras (`token`).

### 🌐 Navegación Pública y Catálogo

| Método | Ruta | Vista / Plantilla | Para qué sirve |
| :--- | :--- | :--- | :--- |
| `GET` | `/` | `index.html` | **Catálogo público:** Página principal que muestra la lista de todos los vehículos aprobados y activos (`publicada`). Incluye barra de navegación que se adapta si el usuario está autenticado. |
| `GET` | `/publications/:id` | `publication_detail.html` | **Ficha del vehículo:** Muestra el carrusel de fotografías, especificaciones completas del auto y la tarjeta de contacto del vendedor (nombre, teléfono, correo y ciudad). |
| `GET` | `/static/*` | Archivos estáticos | **Recursos estáticos:** Sirve estilos CSS (`/static/css/`) e imágenes subidas de los vehículos (`/static/uploads/`). |

---

### 🔐 Autenticación Web

| Método | Ruta | Vista / Plantilla | Para qué sirve |
| :--- | :--- | :--- | :--- |
| `GET` | `/login` | `login.html` | **Formulario de login:** Muestra la pantalla para iniciar sesión. Si el usuario ya está autenticado, lo redirige a su panel (`/seller` o `/admin`). |
| `POST` | `/login` | Redirección | **Procesar login:** Recibe las credenciales enviadas desde el formulario, establece la cookie de sesión y redirige al panel según el rol. |
| `GET` | `/register` | `register.html` | **Formulario de registro:** Muestra la pantalla para registrarse como vendedor. |
| `POST` | `/register` | Redirección | **Procesar registro:** Crea el vendedor en el sistema, guarda la cookie de sesión y redirige a `/seller`. |
| `GET` | `/logout` | Redirección | **Cerrar sesión:** Elimina la cookie `token` y redirige al catálogo público (`/`). |

---

### 💼 Panel del Vendedor (`/seller`)
> *Requiere autenticación con rol `seller` o `admin`.*

| Método | Ruta | Vista / Plantilla | Para qué sirve |
| :--- | :--- | :--- | :--- |
| `GET` | `/seller` | `seller.html` | **Dashboard de vendedor:** Muestra el formulario para publicar nuevos vehículos y la lista de las publicaciones propias con su estado actual (`pendiente`, `publicada`, `vendida`). |
| `POST` | `/seller/publications` | Redirección | **Subir vehículo:** Procesa el formulario con carga de imágenes (multipart) a `/static/uploads/` y crea la publicación en estado pendiente. |
| `POST` | `/seller/publications/sell` | Redirección | **Marcar como vendido:** Permite al vendedor cambiar el estado de su vehículo a `vendida`. |

---

### 🛡️ Panel de Administración (`/admin`)
> *Requiere autenticación con rol `admin`.*

| Método | Ruta | Vista / Plantilla | Para qué sirve |
| :--- | :--- | :--- | :--- |
| `GET` | `/admin` | `admin.html` | **Dashboard de moderación:** Muestra la bandeja de vehículos pendientes de autorización, las publicaciones activas y la tabla de usuarios registrados. |
| `POST` | `/admin/publications/authorize` | Redirección | **Autorizar vehículo:** Aprueba la publicación seleccionada para que sea visible públicamente en el catálogo. |
| `POST` | `/admin/publications/delete` | Redirección | **Eliminar publicación:** Elimina definitivamente un vehículo del sistema. |
| `POST` | `/admin/users/delete` | Redirección | **Eliminar usuario:** Da de baja a un usuario y elimina en cascada todos sus vehículos asociados. |
