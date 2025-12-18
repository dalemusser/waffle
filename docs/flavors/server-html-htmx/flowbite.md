# Flowbite Components in WAFFLE

*Using Flowbite's Tailwind CSS component library with Go templates and HTMX.*

---

## Overview

[Flowbite](https://flowbite.com/) is an open-source UI component library built on Tailwind CSS. It provides 600+ ready-to-use components like buttons, modals, navigation bars, dropdowns, and more—all styled with Tailwind utility classes.

### Why Flowbite with WAFFLE?

| Benefit | Description |
|---------|-------------|
| **No build step for components** | Use via CDN, no npm required |
| **Tailwind-based** | Works with your existing Tailwind setup |
| **HTMX compatible** | Data attributes work alongside hx-* attributes |
| **Accessible** | Components follow WCAG guidelines |
| **Dark mode** | Built-in dark mode support |

---

## Quick Start (CDN)

The fastest way to add Flowbite to a WAFFLE app is via CDN.

### Update layout.gohtml

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ .Title }}</title>

    <!-- Your Tailwind CSS -->
    <link rel="stylesheet" href="/static/css/output.css">

    <!-- Flowbite CSS (optional - only needed for some components) -->
    <link href="https://cdn.jsdelivr.net/npm/flowbite@3.1.2/dist/flowbite.min.css" rel="stylesheet" />
</head>
<body class="bg-gray-100 min-h-screen">
    {{ template "menu" . }}

    <main id="content" class="container mx-auto px-4 py-8">
        {{ template "content" . }}
    </main>

    <!-- HTMX -->
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>

    <!-- Flowbite JS (required for interactive components) -->
    <script src="https://cdn.jsdelivr.net/npm/flowbite@3.1.2/dist/flowbite.min.js"></script>
</body>
</html>
```

**Important**: Place the Flowbite JS script **after** HTMX to ensure both libraries initialize correctly.

---

## Component Categories

### CSS-Only Components (No JavaScript Required)

These components work with just Tailwind classes—no Flowbite JS needed:

- Buttons
- Alerts
- Badges
- Cards
- Breadcrumbs
- Typography
- Pagination
- Progress bars
- Avatars
- Lists

### Interactive Components (JavaScript Required)

These need the Flowbite JS file for functionality:

- Modals
- Dropdowns
- Tooltips
- Popovers
- Tabs
- Accordions
- Carousels
- Datepickers
- Drawers (sidebars)
- Dismissible alerts

---

## Using CSS-Only Components

These work immediately with your Tailwind setup.

### Buttons

```html
{{ define "content" }}
<div class="space-x-4">
    <!-- Primary button -->
    <button type="button" class="text-white bg-blue-700 hover:bg-blue-800 focus:ring-4 focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5">
        Primary
    </button>

    <!-- Secondary button -->
    <button type="button" class="py-2.5 px-5 text-sm font-medium text-gray-900 focus:outline-none bg-white rounded-lg border border-gray-200 hover:bg-gray-100 hover:text-blue-700 focus:z-10 focus:ring-4 focus:ring-gray-100">
        Secondary
    </button>

    <!-- Danger button -->
    <button type="button" class="focus:outline-none text-white bg-red-700 hover:bg-red-800 focus:ring-4 focus:ring-red-300 font-medium rounded-lg text-sm px-5 py-2.5">
        Delete
    </button>
</div>
{{ end }}
```

### Alerts

```html
{{ define "content" }}
<!-- Success alert -->
<div class="p-4 mb-4 text-sm text-green-800 rounded-lg bg-green-50" role="alert">
    <span class="font-medium">Success!</span> Your changes have been saved.
</div>

<!-- Warning alert -->
<div class="p-4 mb-4 text-sm text-yellow-800 rounded-lg bg-yellow-50" role="alert">
    <span class="font-medium">Warning!</span> Please review before continuing.
</div>

<!-- Error alert -->
<div class="p-4 mb-4 text-sm text-red-800 rounded-lg bg-red-50" role="alert">
    <span class="font-medium">Error!</span> {{ .ErrorMessage }}
</div>
{{ end }}
```

### Cards

```html
{{ define "content" }}
<div class="max-w-sm p-6 bg-white border border-gray-200 rounded-lg shadow">
    <h5 class="mb-2 text-2xl font-bold tracking-tight text-gray-900">
        {{ .Card.Title }}
    </h5>
    <p class="mb-3 font-normal text-gray-700">
        {{ .Card.Description }}
    </p>
    <a href="{{ .Card.Link }}"
       hx-get="{{ .Card.Link }}"
       hx-target="#content"
       hx-push-url="true"
       class="inline-flex items-center px-3 py-2 text-sm font-medium text-center text-white bg-blue-700 rounded-lg hover:bg-blue-800 focus:ring-4 focus:outline-none focus:ring-blue-300">
        Read more
        <svg class="rtl:rotate-180 w-3.5 h-3.5 ms-2" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 14 10">
            <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M1 5h12m0 0L9 1m4 4L9 9"/>
        </svg>
    </a>
</div>
{{ end }}
```

### Badges

```html
{{ define "content" }}
<div class="flex items-center space-x-2">
    <span class="bg-blue-100 text-blue-800 text-xs font-medium px-2.5 py-0.5 rounded">Default</span>
    <span class="bg-gray-100 text-gray-800 text-xs font-medium px-2.5 py-0.5 rounded">Dark</span>
    <span class="bg-red-100 text-red-800 text-xs font-medium px-2.5 py-0.5 rounded">Red</span>
    <span class="bg-green-100 text-green-800 text-xs font-medium px-2.5 py-0.5 rounded">Green</span>
    <span class="bg-yellow-100 text-yellow-800 text-xs font-medium px-2.5 py-0.5 rounded">Yellow</span>
</div>

<!-- Status badge in a table -->
<span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full
             {{ if eq .Status "active" }}bg-green-100 text-green-800
             {{ else if eq .Status "pending" }}bg-yellow-100 text-yellow-800
             {{ else }}bg-gray-100 text-gray-800{{ end }}">
    {{ .Status }}
</span>
{{ end }}
```

---

## Using Interactive Components

Interactive components use data attributes to trigger JavaScript behavior.

### Dropdowns

```html
{{ define "content" }}
<button id="dropdownButton" data-dropdown-toggle="dropdown"
        class="text-white bg-blue-700 hover:bg-blue-800 focus:ring-4 focus:outline-none focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center inline-flex items-center"
        type="button">
    Actions
    <svg class="w-2.5 h-2.5 ms-3" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 10 6">
        <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m1 1 4 4 4-4"/>
    </svg>
</button>

<!-- Dropdown menu -->
<div id="dropdown" class="z-10 hidden bg-white divide-y divide-gray-100 rounded-lg shadow w-44">
    <ul class="py-2 text-sm text-gray-700" aria-labelledby="dropdownButton">
        <li>
            <a href="/users/{{ .User.ID }}/edit"
               hx-get="/users/{{ .User.ID }}/edit"
               hx-target="#content"
               class="block px-4 py-2 hover:bg-gray-100">
                Edit
            </a>
        </li>
        <li>
            <a href="/users/{{ .User.ID }}/settings"
               hx-get="/users/{{ .User.ID }}/settings"
               hx-target="#content"
               class="block px-4 py-2 hover:bg-gray-100">
                Settings
            </a>
        </li>
        <li>
            <button hx-delete="/users/{{ .User.ID }}"
                    hx-target="#content"
                    hx-confirm="Delete this user?"
                    class="block w-full text-left px-4 py-2 text-red-600 hover:bg-gray-100">
                Delete
            </button>
        </li>
    </ul>
</div>
{{ end }}
```

### Modals

```html
{{ define "content" }}
<!-- Trigger button -->
<button data-modal-target="deleteModal" data-modal-toggle="deleteModal"
        class="text-white bg-red-700 hover:bg-red-800 focus:ring-4 focus:outline-none focus:ring-red-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center"
        type="button">
    Delete Item
</button>

<!-- Modal -->
<div id="deleteModal" tabindex="-1" aria-hidden="true"
     class="hidden overflow-y-auto overflow-x-hidden fixed top-0 right-0 left-0 z-50 justify-center items-center w-full md:inset-0 h-[calc(100%-1rem)] max-h-full">
    <div class="relative p-4 w-full max-w-md max-h-full">
        <div class="relative bg-white rounded-lg shadow">
            <!-- Modal header -->
            <div class="flex items-center justify-between p-4 md:p-5 border-b rounded-t">
                <h3 class="text-xl font-semibold text-gray-900">
                    Confirm Delete
                </h3>
                <button type="button" data-modal-hide="deleteModal"
                        class="text-gray-400 bg-transparent hover:bg-gray-200 hover:text-gray-900 rounded-lg text-sm w-8 h-8 ms-auto inline-flex justify-center items-center">
                    <svg class="w-3 h-3" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 14 14">
                        <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m1 1 6 6m0 0 6 6M7 7l6-6M7 7l-6 6"/>
                    </svg>
                    <span class="sr-only">Close modal</span>
                </button>
            </div>
            <!-- Modal body -->
            <div class="p-4 md:p-5 space-y-4">
                <p class="text-base leading-relaxed text-gray-500">
                    Are you sure you want to delete "{{ .Item.Name }}"? This action cannot be undone.
                </p>
            </div>
            <!-- Modal footer -->
            <div class="flex items-center p-4 md:p-5 border-t border-gray-200 rounded-b">
                <button hx-delete="/items/{{ .Item.ID }}"
                        hx-target="#content"
                        data-modal-hide="deleteModal"
                        class="text-white bg-red-700 hover:bg-red-800 focus:ring-4 focus:outline-none focus:ring-red-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center">
                    Yes, delete
                </button>
                <button data-modal-hide="deleteModal" type="button"
                        class="py-2.5 px-5 ms-3 text-sm font-medium text-gray-900 focus:outline-none bg-white rounded-lg border border-gray-200 hover:bg-gray-100 hover:text-blue-700 focus:z-10 focus:ring-4 focus:ring-gray-100">
                    Cancel
                </button>
            </div>
        </div>
    </div>
</div>
{{ end }}
```

### Tabs

```html
{{ define "content" }}
<div class="mb-4 border-b border-gray-200">
    <ul class="flex flex-wrap -mb-px text-sm font-medium text-center" id="tabs" data-tabs-toggle="#tabContent" role="tablist">
        <li class="me-2" role="presentation">
            <button class="inline-block p-4 border-b-2 rounded-t-lg" id="profile-tab" data-tabs-target="#profile" type="button" role="tab" aria-controls="profile" aria-selected="false">
                Profile
            </button>
        </li>
        <li class="me-2" role="presentation">
            <button class="inline-block p-4 border-b-2 rounded-t-lg hover:text-gray-600 hover:border-gray-300" id="settings-tab" data-tabs-target="#settings" type="button" role="tab" aria-controls="settings" aria-selected="false">
                Settings
            </button>
        </li>
        <li role="presentation">
            <button class="inline-block p-4 border-b-2 rounded-t-lg hover:text-gray-600 hover:border-gray-300" id="contacts-tab" data-tabs-target="#contacts" type="button" role="tab" aria-controls="contacts" aria-selected="false">
                Contacts
            </button>
        </li>
    </ul>
</div>
<div id="tabContent">
    <div class="hidden p-4 rounded-lg bg-gray-50" id="profile" role="tabpanel" aria-labelledby="profile-tab">
        {{ template "profile_content" . }}
    </div>
    <div class="hidden p-4 rounded-lg bg-gray-50" id="settings" role="tabpanel" aria-labelledby="settings-tab">
        {{ template "settings_content" . }}
    </div>
    <div class="hidden p-4 rounded-lg bg-gray-50" id="contacts" role="tabpanel" aria-labelledby="contacts-tab">
        {{ template "contacts_content" . }}
    </div>
</div>
{{ end }}
```

### Dismissible Alerts

```html
{{ define "content" }}
{{ if .SuccessMessage }}
<div id="alert-success" class="flex items-center p-4 mb-4 text-green-800 rounded-lg bg-green-50" role="alert">
    <svg class="flex-shrink-0 w-4 h-4" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="currentColor" viewBox="0 0 20 20">
        <path d="M10 .5a9.5 9.5 0 1 0 9.5 9.5A9.51 9.51 0 0 0 10 .5Zm3.707 8.207-4 4a1 1 0 0 1-1.414 0l-2-2a1 1 0 0 1 1.414-1.414L9 10.586l3.293-3.293a1 1 0 0 1 1.414 1.414Z"/>
    </svg>
    <span class="sr-only">Success</span>
    <div class="ms-3 text-sm font-medium">
        {{ .SuccessMessage }}
    </div>
    <button type="button" class="ms-auto -mx-1.5 -my-1.5 bg-green-50 text-green-500 rounded-lg focus:ring-2 focus:ring-green-400 p-1.5 hover:bg-green-200 inline-flex items-center justify-center h-8 w-8" data-dismiss-target="#alert-success" aria-label="Close">
        <span class="sr-only">Close</span>
        <svg class="w-3 h-3" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 14 14">
            <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m1 1 6 6m0 0 6 6M7 7l6-6M7 7l-6 6"/>
        </svg>
    </button>
</div>
{{ end }}
{{ end }}
```

---

## HTMX + Flowbite Patterns

### Reinitializing After HTMX Swaps

When HTMX swaps content containing Flowbite components, you need to reinitialize them:

```html
<!-- In layout.gohtml -->
<script>
document.body.addEventListener('htmx:afterSwap', function(evt) {
    // Reinitialize all Flowbite components
    if (typeof initFlowbite === 'function') {
        initFlowbite();
    }
});
</script>
```

### Modal Forms with HTMX

Load modal content dynamically:

```html
{{ define "content" }}
<!-- Button triggers HTMX load into modal -->
<button hx-get="/users/new"
        hx-target="#modal-content"
        hx-trigger="click"
        data-modal-target="formModal"
        data-modal-toggle="formModal"
        class="text-white bg-blue-700 hover:bg-blue-800 font-medium rounded-lg text-sm px-5 py-2.5">
    Add User
</button>

<!-- Modal with dynamic content area -->
<div id="formModal" tabindex="-1" aria-hidden="true"
     class="hidden overflow-y-auto overflow-x-hidden fixed top-0 right-0 left-0 z-50 justify-center items-center w-full md:inset-0 h-[calc(100%-1rem)] max-h-full">
    <div class="relative p-4 w-full max-w-md max-h-full">
        <div class="relative bg-white rounded-lg shadow">
            <div id="modal-content">
                <!-- HTMX loads form content here -->
                <div class="p-4 text-center">
                    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-700 mx-auto"></div>
                    <p class="mt-2 text-gray-500">Loading...</p>
                </div>
            </div>
        </div>
    </div>
</div>
{{ end }}
```

### Dropdown with HTMX Actions

```html
{{ define "user_row" }}
<tr id="user-{{ .ID }}" class="bg-white border-b hover:bg-gray-50">
    <td class="px-6 py-4 font-medium text-gray-900">{{ .Name }}</td>
    <td class="px-6 py-4">{{ .Email }}</td>
    <td class="px-6 py-4">
        <button id="dropdown-{{ .ID }}" data-dropdown-toggle="dropdown-menu-{{ .ID }}"
                class="inline-flex items-center p-2 text-sm font-medium text-gray-500 bg-white rounded-lg hover:bg-gray-100 focus:ring-4 focus:outline-none focus:ring-gray-200"
                type="button">
            <svg class="w-5 h-5" aria-hidden="true" fill="currentColor" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
                <path d="M6 10a2 2 0 11-4 0 2 2 0 014 0zM12 10a2 2 0 11-4 0 2 2 0 014 0zM16 12a2 2 0 100-4 2 2 0 000 4z"/>
            </svg>
        </button>
        <div id="dropdown-menu-{{ .ID }}" class="z-10 hidden bg-white divide-y divide-gray-100 rounded-lg shadow w-44">
            <ul class="py-2 text-sm text-gray-700">
                <li>
                    <a hx-get="/users/{{ .ID }}/edit"
                       hx-target="#content"
                       hx-push-url="true"
                       class="block px-4 py-2 hover:bg-gray-100 cursor-pointer">
                        Edit
                    </a>
                </li>
                <li>
                    <button hx-delete="/users/{{ .ID }}"
                            hx-target="#user-{{ .ID }}"
                            hx-swap="outerHTML swap:1s"
                            hx-confirm="Delete {{ .Name }}?"
                            class="block w-full text-left px-4 py-2 text-red-600 hover:bg-gray-100">
                        Delete
                    </button>
                </li>
            </ul>
        </div>
    </td>
</tr>
{{ end }}
```

---

## Navigation Components

### Navbar

```html
{{ define "menu" }}
<nav class="bg-white border-gray-200">
    <div class="max-w-screen-xl flex flex-wrap items-center justify-between mx-auto p-4">
        <a href="/" class="flex items-center space-x-3 rtl:space-x-reverse">
            <span class="self-center text-2xl font-semibold whitespace-nowrap">{{ .AppName }}</span>
        </a>
        <button data-collapse-toggle="navbar-default" type="button"
                class="inline-flex items-center p-2 w-10 h-10 justify-center text-sm text-gray-500 rounded-lg md:hidden hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-200"
                aria-controls="navbar-default" aria-expanded="false">
            <span class="sr-only">Open main menu</span>
            <svg class="w-5 h-5" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 17 14">
                <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M1 1h15M1 7h15M1 13h15"/>
            </svg>
        </button>
        <div class="hidden w-full md:block md:w-auto" id="navbar-default">
            <ul class="font-medium flex flex-col p-4 md:p-0 mt-4 border border-gray-100 rounded-lg bg-gray-50 md:flex-row md:space-x-8 rtl:space-x-reverse md:mt-0 md:border-0 md:bg-white">
                <li>
                    <a href="/"
                       hx-get="/"
                       hx-target="#content"
                       hx-push-url="true"
                       class="block py-2 px-3 text-gray-900 rounded hover:bg-gray-100 md:hover:bg-transparent md:border-0 md:hover:text-blue-700 md:p-0">
                        Home
                    </a>
                </li>
                <li>
                    <a href="/users"
                       hx-get="/users"
                       hx-target="#content"
                       hx-push-url="true"
                       class="block py-2 px-3 text-gray-900 rounded hover:bg-gray-100 md:hover:bg-transparent md:border-0 md:hover:text-blue-700 md:p-0">
                        Users
                    </a>
                </li>
                <li>
                    <a href="/settings"
                       hx-get="/settings"
                       hx-target="#content"
                       hx-push-url="true"
                       class="block py-2 px-3 text-gray-900 rounded hover:bg-gray-100 md:hover:bg-transparent md:border-0 md:hover:text-blue-700 md:p-0">
                        Settings
                    </a>
                </li>
            </ul>
        </div>
    </div>
</nav>
{{ end }}
```

### Sidebar Navigation

```html
{{ define "sidebar" }}
<aside id="sidebar" class="fixed top-0 left-0 z-40 w-64 h-screen transition-transform -translate-x-full sm:translate-x-0" aria-label="Sidebar">
    <div class="h-full px-3 py-4 overflow-y-auto bg-gray-50">
        <ul class="space-y-2 font-medium">
            <li>
                <a href="/dashboard"
                   hx-get="/dashboard"
                   hx-target="#content"
                   hx-push-url="true"
                   class="flex items-center p-2 text-gray-900 rounded-lg hover:bg-gray-100 group">
                    <svg class="w-5 h-5 text-gray-500 transition duration-75 group-hover:text-gray-900" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="currentColor" viewBox="0 0 22 21">
                        <path d="M16.975 11H10V4.025a1 1 0 0 0-1.066-.998 8.5 8.5 0 1 0 9.039 9.039.999.999 0 0 0-1-1.066h.002Z"/>
                        <path d="M12.5 0c-.157 0-.311.01-.565.027A1 1 0 0 0 11 1.02V10h8.975a1 1 0 0 0 1-.935c.013-.188.028-.374.028-.565A8.51 8.51 0 0 0 12.5 0Z"/>
                    </svg>
                    <span class="ms-3">Dashboard</span>
                </a>
            </li>
            <li>
                <button type="button" class="flex items-center w-full p-2 text-base text-gray-900 transition duration-75 rounded-lg group hover:bg-gray-100" aria-controls="dropdown-users" data-collapse-toggle="dropdown-users">
                    <svg class="flex-shrink-0 w-5 h-5 text-gray-500 transition duration-75 group-hover:text-gray-900" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="currentColor" viewBox="0 0 20 18">
                        <path d="M14 2a3.963 3.963 0 0 0-1.4.267 6.439 6.439 0 0 1-1.331 6.638A4 4 0 1 0 14 2Zm1 9h-1.264A6.957 6.957 0 0 1 15 15v2a2.97 2.97 0 0 1-.184 1H19a1 1 0 0 0 1-1v-1a5.006 5.006 0 0 0-5-5ZM6.5 9a4.5 4.5 0 1 0 0-9 4.5 4.5 0 0 0 0 9ZM8 10H5a5.006 5.006 0 0 0-5 5v2a1 1 0 0 0 1 1h11a1 1 0 0 0 1-1v-2a5.006 5.006 0 0 0-5-5Z"/>
                    </svg>
                    <span class="flex-1 ms-3 text-left rtl:text-right whitespace-nowrap">Users</span>
                    <svg class="w-3 h-3" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 10 6">
                        <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m1 1 4 4 4-4"/>
                    </svg>
                </button>
                <ul id="dropdown-users" class="hidden py-2 space-y-2">
                    <li>
                        <a href="/users"
                           hx-get="/users"
                           hx-target="#content"
                           hx-push-url="true"
                           class="flex items-center w-full p-2 text-gray-900 transition duration-75 rounded-lg pl-11 group hover:bg-gray-100">
                            All Users
                        </a>
                    </li>
                    <li>
                        <a href="/users/new"
                           hx-get="/users/new"
                           hx-target="#content"
                           hx-push-url="true"
                           class="flex items-center w-full p-2 text-gray-900 transition duration-75 rounded-lg pl-11 group hover:bg-gray-100">
                            Add User
                        </a>
                    </li>
                </ul>
            </li>
        </ul>
    </div>
</aside>
{{ end }}
```

---

## Forms with Flowbite Styling

### Login Form

```html
{{ define "content" }}
<div class="flex min-h-full flex-col justify-center px-6 py-12 lg:px-8">
    <div class="sm:mx-auto sm:w-full sm:max-w-sm">
        <h2 class="mt-10 text-center text-2xl font-bold leading-9 tracking-tight text-gray-900">
            Sign in to your account
        </h2>
    </div>

    <div class="mt-10 sm:mx-auto sm:w-full sm:max-w-sm">
        <form class="space-y-6" hx-post="/login" hx-target="#content">
            <div>
                <label for="email" class="block mb-2 text-sm font-medium text-gray-900">Email address</label>
                <input type="email" name="email" id="email" autocomplete="email" required
                       class="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5"
                       placeholder="name@example.com">
            </div>

            <div>
                <label for="password" class="block mb-2 text-sm font-medium text-gray-900">Password</label>
                <input type="password" name="password" id="password" autocomplete="current-password" required
                       class="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5"
                       placeholder="••••••••">
            </div>

            <div class="flex items-center justify-between">
                <div class="flex items-start">
                    <div class="flex items-center h-5">
                        <input id="remember" type="checkbox" name="remember"
                               class="w-4 h-4 border border-gray-300 rounded bg-gray-50 focus:ring-3 focus:ring-blue-300">
                    </div>
                    <label for="remember" class="ms-2 text-sm font-medium text-gray-900">Remember me</label>
                </div>
                <a href="/forgot-password" class="text-sm font-medium text-blue-600 hover:underline">Forgot password?</a>
            </div>

            <button type="submit"
                    class="w-full text-white bg-blue-700 hover:bg-blue-800 focus:ring-4 focus:outline-none focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center">
                Sign in
            </button>
        </form>
    </div>
</div>
{{ end }}
```

---

## Dark Mode Support

Flowbite supports dark mode via Tailwind's `dark:` variant.

### Enable Dark Mode in Tailwind

```javascript
// tailwind.config.js
module.exports = {
  darkMode: 'class', // or 'media' for system preference
  content: [
    './internal/app/resources/templates/**/*.gohtml',
    './internal/app/features/**/templates/**/*.gohtml',
  ],
  // ...
}
```

### Dark Mode Toggle

```html
{{ define "dark_mode_toggle" }}
<button id="theme-toggle" type="button"
        class="text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 focus:outline-none focus:ring-4 focus:ring-gray-200 dark:focus:ring-gray-700 rounded-lg text-sm p-2.5">
    <svg id="theme-toggle-dark-icon" class="hidden w-5 h-5" fill="currentColor" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
        <path d="M17.293 13.293A8 8 0 016.707 2.707a8.001 8.001 0 1010.586 10.586z"/>
    </svg>
    <svg id="theme-toggle-light-icon" class="hidden w-5 h-5" fill="currentColor" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
        <path d="M10 2a1 1 0 011 1v1a1 1 0 11-2 0V3a1 1 0 011-1zm4 8a4 4 0 11-8 0 4 4 0 018 0zm-.464 4.95l.707.707a1 1 0 001.414-1.414l-.707-.707a1 1 0 00-1.414 1.414zm2.12-10.607a1 1 0 010 1.414l-.706.707a1 1 0 11-1.414-1.414l.707-.707a1 1 0 011.414 0zM17 11a1 1 0 100-2h-1a1 1 0 100 2h1zm-7 4a1 1 0 011 1v1a1 1 0 11-2 0v-1a1 1 0 011-1zM5.05 6.464A1 1 0 106.465 5.05l-.708-.707a1 1 0 00-1.414 1.414l.707.707zm1.414 8.486l-.707.707a1 1 0 01-1.414-1.414l.707-.707a1 1 0 011.414 1.414zM4 11a1 1 0 100-2H3a1 1 0 000 2h1z" fill-rule="evenodd" clip-rule="evenodd"/>
    </svg>
</button>

<script>
var themeToggleDarkIcon = document.getElementById('theme-toggle-dark-icon');
var themeToggleLightIcon = document.getElementById('theme-toggle-light-icon');

// Change the icons inside the button based on previous settings
if (localStorage.getItem('color-theme') === 'dark' || (!('color-theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    themeToggleLightIcon.classList.remove('hidden');
} else {
    themeToggleDarkIcon.classList.remove('hidden');
}

var themeToggleBtn = document.getElementById('theme-toggle');

themeToggleBtn.addEventListener('click', function() {
    themeToggleDarkIcon.classList.toggle('hidden');
    themeToggleLightIcon.classList.toggle('hidden');

    if (localStorage.getItem('color-theme')) {
        if (localStorage.getItem('color-theme') === 'light') {
            document.documentElement.classList.add('dark');
            localStorage.setItem('color-theme', 'dark');
        } else {
            document.documentElement.classList.remove('dark');
            localStorage.setItem('color-theme', 'light');
        }
    } else {
        if (document.documentElement.classList.contains('dark')) {
            document.documentElement.classList.remove('dark');
            localStorage.setItem('color-theme', 'light');
        } else {
            document.documentElement.classList.add('dark');
            localStorage.setItem('color-theme', 'dark');
        }
    }
});
</script>
{{ end }}
```

---

## Troubleshooting

### Components Not Working After HTMX Swap

**Cause**: Flowbite initializes on page load, but new elements need reinitialization.

**Solution**: Add the reinitialize script to your layout:

```html
<script>
document.body.addEventListener('htmx:afterSwap', function(evt) {
    if (typeof initFlowbite === 'function') {
        initFlowbite();
    }
});
</script>
```

### Dropdown Appears Behind Other Elements

**Cause**: Z-index stacking issue.

**Solution**: Ensure dropdown has high z-index and parent elements don't have `overflow: hidden`:

```html
<div id="dropdown" class="z-50 hidden ...">
```

### Modal Not Closing

**Cause**: Multiple modals with same ID or event conflicts.

**Solution**: Ensure unique IDs and use `data-modal-hide` on close buttons:

```html
<button data-modal-hide="myModal" type="button" class="...">Close</button>
```

---

## See Also

- [Flowbite Documentation](https://flowbite.com/docs/getting-started/introduction/)
- [Flowbite Components](https://flowbite.com/docs/components/accordion/)
- [Tailwind CSS Setup](./tailwind-css.md)
- [HTMX Integration](./htmx-integration.md)
- [Templates and Views](./templates-and-views.md)

---

[← Back to Frontend Documentation](./README.md)
