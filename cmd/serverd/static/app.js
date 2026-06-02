// S3Go - Single Page Application Javascript

// --- 1. Global State ---
let connections = [];
let activeConnection = null;
let currentPrefix = "";
let selectedFiles = new Set();
let activeUploads = {}; // Tracks active upload XHR objects

// --- 2. Initialize App ---
document.addEventListener("DOMContentLoaded", async () => {
    initEventListeners();
    await fetchConnections();
    restoreStateFromURL();
});

// --- 3. Event Listeners ---
function initEventListeners() {
    // Modal connection
    document.getElementById("btn-add-conn").addEventListener("click", () => {
        const form = document.getElementById("form-connection");
        form.setAttribute("data-mode", "create");
        form.removeAttribute("data-id");
        document.getElementById("modal-conn-title").innerText = "Add New S3 Connection";
        
        const secretInput = document.getElementById("conn-secret-key");
        secretInput.setAttribute("required", "required");
        secretInput.setAttribute("placeholder", "Enter Secret Key");
        secretInput.value = "";
        
        form.reset();
        openModal("modal-connection");
    });
    document.getElementById("form-connection").addEventListener("submit", handleSaveConnection);

    // Modal folder
    document.getElementById("btn-create-folder").addEventListener("click", () => openModal("modal-folder"));
    document.getElementById("form-folder").addEventListener("submit", handleCreateFolder);

    // File input change
    const uploadBtn = document.getElementById("btn-upload");
    const fileInput = document.createElement("input");
    fileInput.type = "file";
    fileInput.multiple = true;
    fileInput.style.display = "none";
    document.body.appendChild(fileInput);

    uploadBtn.addEventListener("click", () => fileInput.click());
    fileInput.addEventListener("change", (e) => {
        handleFileUploads(e.target.files);
        fileInput.value = ""; // Reset value
    });

    // Drag & Drop
    const dropzone = document.getElementById("dropzone-overlay");
    window.addEventListener("dragenter", (e) => {
        if (activeConnection) {
            e.preventDefault();
            dropzone.classList.add("active");
        }
    });

    dropzone.addEventListener("dragover", (e) => {
        e.preventDefault();
    });

    dropzone.addEventListener("dragleave", (e) => {
        e.preventDefault();
        // Only remove active if mouse leaves screen or dropzone
        if (e.relatedTarget === null) {
            dropzone.classList.remove("active");
        }
    });

    dropzone.addEventListener("drop", (e) => {
        e.preventDefault();
        dropzone.classList.remove("active");
        if (activeConnection) {
            handleFileUploads(e.dataTransfer.files);
        }
    });

    // Search bar input & clear
    const searchInput = document.getElementById("search-input");
    const clearSearchBtn = document.getElementById("btn-clear-search");
    
    // Live search with debounce
    let searchDebounce;
    searchInput.addEventListener("input", (e) => {
        const val = e.target.value.trim();
        clearSearchBtn.style.display = val ? "block" : "none";

        clearTimeout(searchDebounce);
        searchDebounce = setTimeout(() => {
            clearSelection();
            fetchFiles(val);
        }, 350); // 350ms debounce
    });

    clearSearchBtn.addEventListener("click", () => {
        searchInput.value = "";
        clearSearchBtn.style.display = "none";
        clearSelection();
        fetchFiles("");
    });

    // Select-all files checkbox
    const selectAllCheckbox = document.getElementById("select-all-checkbox");
    selectAllCheckbox.addEventListener("change", (e) => {
        const isChecked = e.target.checked;
        const checkboxes = document.querySelectorAll(".file-checkbox");
        checkboxes.forEach(cb => {
            cb.checked = isChecked;
            const row = cb.closest(".file-row");
            const key = cb.getAttribute("data-key");
            if (isChecked) {
                row.classList.add("selected");
                selectedFiles.add(key);
            } else {
                row.classList.remove("selected");
                selectedFiles.delete(key);
            }
        });
        updateBulkActionsBar();
    });

    // Bulk delete buttons
    document.getElementById("btn-bulk-delete").addEventListener("click", () => {
        openDeleteConfirmation(Array.from(selectedFiles));
    });
    
    document.getElementById("btn-clear-selection").addEventListener("click", clearSelection);

    // Close upload pane
    document.getElementById("btn-close-upload-pane").addEventListener("click", () => {
        document.getElementById("upload-progress-pane").style.display = "none";
    });

    // Share link copied toast
    document.getElementById("btn-copy-url").addEventListener("click", () => {
        const urlInput = document.getElementById("presigned-url-output");
        urlInput.select();
        document.execCommand("copy");
        showToast("success", "Link copied to clipboard!");
    });

    // Global ESC key listener to close modals
    document.addEventListener("keydown", (e) => {
        if (e.key === "Escape") {
            const activeModals = document.querySelectorAll(".modal.active");
            activeModals.forEach(modal => closeModal(modal.id));
        }
    });

    // Browser Back/Forward navigation popstate sync
    window.addEventListener("popstate", (event) => {
        const state = event.state;
        const connId = state ? state.connectionId : null;
        const prefix = state ? state.prefix : "";
        
        if (connId) {
            activeConnection = connId;
            currentPrefix = prefix;
            
            // Refresh connection sidebar style to match activeConnection
            renderConnections();
            
            const activeConn = connections.find(c => c.id === connId);
            if (activeConn) {
                document.getElementById("active-connection-title").innerText = activeConn.name;
                document.getElementById("active-bucket-name").innerText = activeConn.bucket || "All Buckets";
                
                // Refresh buttons visibility
                const activeConnInList = connections.find(c => c.id === activeConnection);
                const showBtns = activeConnInList && (activeConnInList.bucket !== "" || prefix !== "");
                document.getElementById("btn-upload").style.display = showBtns ? "inline-flex" : "none";
                document.getElementById("btn-create-folder").style.display = showBtns ? "inline-flex" : "none";
                document.getElementById("nav-bar").style.display = "flex";
                document.getElementById("active-bucket-badge").style.display = activeConn.bucket ? "flex" : "none";
                document.getElementById("placeholder-select-conn").style.display = "none";
                
                clearSelection();
                fetchFiles("");
            }
        } else {
            // Reset to main welcome screen
            activeConnection = null;
            currentPrefix = "";
            renderConnections();
            
            document.getElementById("btn-upload").style.display = "none";
            document.getElementById("btn-create-folder").style.display = "none";
            document.getElementById("nav-bar").style.display = "none";
            document.getElementById("active-bucket-badge").style.display = "none";
            document.getElementById("active-connection-title").innerText = "No Active Connection";
            document.getElementById("placeholder-select-conn").style.display = "flex";
            document.getElementById("files-container").style.display = "none";
        }
    });
}

// --- 4. Connection Management ---
async function fetchConnections() {
    try {
        const res = await fetch("/api/connections");
        if (!res.ok) throw new Error("Failed to load connection list");
        connections = await res.json();
        renderConnections();
    } catch (err) {
        showToast("error", err.message);
    }
}

function renderConnections() {
    const list = document.getElementById("conn-list");
    list.innerHTML = "";

    if (!connections || connections.length === 0) {
        list.innerHTML = `<div class="connections-empty">No connections saved. Create a new connection to start!</div>`;
        return;
    }

    connections.forEach(conn => {
        const item = document.createElement("div");
        item.className = `conn-item ${activeConnection === conn.id ? "active" : ""}`;
        item.onclick = () => selectConnection(conn.id);

        item.innerHTML = `
            <div class="conn-info">
                <i class="fa-solid fa-hard-drive conn-icon"></i>
                <div class="conn-details">
                    <span class="conn-name">${conn.name}</span>
                    <span class="conn-bucket">${conn.bucket ? conn.bucket : "All Buckets"} (${conn.region})</span>
                </div>
            </div>
            <div class="conn-actions">
                <button class="btn-conn-edit" onclick="handleEditConnection(event, '${conn.id}')" title="Edit connection">
                    <i class="fa-regular fa-pen-to-square"></i>
                </button>
                <button class="btn-conn-delete" onclick="handleDeleteConnection(event, '${conn.id}', '${conn.name}')" title="Delete connection">
                    <i class="fa-regular fa-trash-can"></i>
                </button>
            </div>
        `;
        list.appendChild(item);
    });
}

function handleEditConnection(e, id) {
    e.stopPropagation(); // Avoid triggering connection selection
    const conn = connections.find(c => c.id === id);
    if (!conn) return;

    const form = document.getElementById("form-connection");
    form.setAttribute("data-mode", "edit");
    form.setAttribute("data-id", id);
    document.getElementById("modal-conn-title").innerText = "Edit S3 Connection";

    // Populate inputs
    document.getElementById("conn-name").value = conn.name || "";
    document.getElementById("conn-access-key").value = conn.access_key || "";
    document.getElementById("conn-region").value = conn.region || "";
    document.getElementById("conn-bucket").value = conn.bucket || "";
    
    // Secret key is optional on edit
    const secretKeyInput = document.getElementById("conn-secret-key");
    secretKeyInput.removeAttribute("required");
    secretKeyInput.value = "";
    secretKeyInput.setAttribute("placeholder", "•••••••• (Leave blank to keep unchanged)");

    // Clear verification messages
    document.getElementById("conn-verify-status").innerHTML = "";

    openModal("modal-connection");
}

async function handleSaveConnection(e) {
    e.preventDefault();
    
    const form = document.getElementById("form-connection");
    const mode = form.getAttribute("data-mode");
    const id = form.getAttribute("data-id");

    const saveBtn = document.getElementById("btn-save-conn-submit");
    const btnText = document.getElementById("save-btn-text");
    const spinner = document.getElementById("save-btn-spinner");
    const statusLabel = document.getElementById("conn-verify-status");

    // Disable button & show spinner
    saveBtn.disabled = true;
    btnText.innerText = mode === "edit" ? "Updating..." : "Testing & saving...";
    spinner.style.display = "inline-block";
    statusLabel.innerHTML = "";

    const payload = {
        name: document.getElementById("conn-name").value.trim(),
        access_key: document.getElementById("conn-access-key").value.trim(),
        secret_key: document.getElementById("conn-secret-key").value.trim(),
        region: document.getElementById("conn-region").value.trim(),
        bucket: document.getElementById("conn-bucket").value.trim()
    };

    try {
        let res;
        if (mode === "edit") {
            res = await fetch(`/api/connections/${id}`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(payload)
            });
        } else {
            res = await fetch("/api/connections", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(payload)
            });
        }

        const data = await res.json();
        
        if (!res.ok) {
            throw new Error(data.error || "AWS verification failed");
        }

        if (mode === "edit") {
            showToast("success", "Connection updated successfully!");
        } else {
            if (payload.bucket) {
                showToast("success", `Successfully connected to bucket: ${payload.bucket}`);
            } else {
                showToast("success", "Successfully connected to AWS S3!");
            }
        }
        closeModal("modal-connection");
        form.reset();
        
        // Refresh connection list
        await fetchConnections();
        
        if (mode === "edit") {
            // If the edited connection was active, re-select to refresh header/region badge
            if (activeConnection === id) {
                selectConnection(id);
            }
        } else {
            // Select the new connection
            selectConnection(data.id);
        }
    } catch (err) {
        statusLabel.innerHTML = `<span class="verify-error"><i class="fa-solid fa-triangle-exclamation"></i> Error: ${err.message}</span>`;
        showToast("error", err.message);
    } finally {
        saveBtn.disabled = false;
        btnText.innerText = mode === "edit" ? "Update & Save" : "Test & Save";
        spinner.style.display = "none";
    }
}

async function handleDeleteConnection(e, id, name) {
    e.stopPropagation(); // Avoid triggering connection selection
    
    if (!confirm(`Are you sure you want to delete connection "${name}"?`)) {
        return;
    }

    try {
        const res = await fetch(`/api/connections/${id}`, { method: "DELETE" });
        if (!res.ok) throw new Error("Failed to delete connection");
        
        showToast("success", `Connection "${name}" deleted`);
        
        if (activeConnection === id) {
            activeConnection = null;
            currentPrefix = "";
            document.getElementById("btn-upload").style.display = "none";
            document.getElementById("btn-create-folder").style.display = "none";
            document.getElementById("nav-bar").style.display = "none";
            document.getElementById("active-bucket-badge").style.display = "none";
            document.getElementById("active-connection-title").innerText = "No Active Connection";
            document.getElementById("placeholder-select-conn").style.display = "flex";
            document.getElementById("files-container").style.display = "none";
            updateURLAndHistory(true);
        }
        
        fetchConnections();
    } catch (err) {
        showToast("error", err.message);
    }
}

function selectConnection(id) {
    activeConnection = id;
    currentPrefix = "";
    
    // Update active class in list
    const items = document.querySelectorAll(".conn-item");
    items.forEach(item => item.classList.remove("active"));
    
    // Refresh connections sidebar to reflect active change
    renderConnections();

    const activeConn = connections.find(c => c.id === id);
    if (activeConn) {
        document.getElementById("active-connection-title").innerText = activeConn.name;
        document.getElementById("active-bucket-name").innerText = activeConn.bucket || "All Buckets";
        
        // Show file navigation elements
        document.getElementById("btn-upload").style.display = activeConn.bucket ? "inline-flex" : "none";
        document.getElementById("btn-create-folder").style.display = activeConn.bucket ? "inline-flex" : "none";
        document.getElementById("nav-bar").style.display = "flex";
        document.getElementById("active-bucket-badge").style.display = activeConn.bucket ? "flex" : "none";
        document.getElementById("placeholder-select-conn").style.display = "none";

        clearSelection();
        fetchFiles("");
        updateURLAndHistory(true);
    }
}

// --- 5. File Explorer ---
async function fetchFiles(search = "") {
    if (!activeConnection) return;

    const listContainer = document.getElementById("files-container");
    const loading = document.getElementById("viewport-loading");
    const emptyPlaceholder = document.getElementById("placeholder-empty-bucket");
    const filesList = document.getElementById("files-list");
    const selectAllCheckbox = document.getElementById("select-all-checkbox");

    listContainer.style.display = "none";
    emptyPlaceholder.style.display = "none";
    loading.style.display = "flex";
    selectAllCheckbox.checked = false;

    try {
        let url = `/api/connections/${activeConnection}/files?prefix=${encodeURIComponent(currentPrefix)}`;
        if (search) {
            url += `&search=${encodeURIComponent(search)}`;
        }

        const res = await fetch(url);
        if (!res.ok) {
            const data = await res.json();
            throw new Error(data.error || "Failed to retrieve file list");
        }

        const items = await res.json();
        loading.style.display = "none";
        
        renderBreadcrumbs(search);

        // Hide upload and create folder buttons at root level in all-buckets mode
        const activeConn = connections.find(c => c.id === activeConnection);
        if (activeConn && activeConn.bucket === "" && currentPrefix === "") {
            document.getElementById("btn-upload").style.display = "none";
            document.getElementById("btn-create-folder").style.display = "none";
            document.getElementById("active-bucket-badge").style.display = "none";
        } else {
            document.getElementById("btn-upload").style.display = "inline-flex";
            document.getElementById("btn-create-folder").style.display = "inline-flex";
            document.getElementById("active-bucket-badge").style.display = "flex";
            if (activeConn) {
                document.getElementById("active-bucket-name").innerText = activeConn.bucket || currentPrefix.split("/")[0];
            }
        }

        if (!items || items.length === 0) {
            emptyPlaceholder.style.display = "flex";
            return;
        }

        listContainer.style.display = "flex";
        filesList.innerHTML = "";

        items.forEach(item => {
            const row = document.createElement("div");
            row.className = `file-row ${item.is_dir ? "directory" : ""} ${selectedFiles.has(item.key) ? "selected" : ""}`;
            
            // File size display
            let sizeStr = "-";
            if (!item.is_dir) {
                sizeStr = formatBytes(item.size);
            }

            // Modified date display
            let dateStr = "-";
            if (!item.is_dir && item.last_modified) {
                const date = new Date(item.last_modified);
                dateStr = date.toLocaleString("en-US");
            }

            // Icon class
            const iconClass = item.is_dir ? "fa-solid fa-folder" : getFileIcon(item.name);

            row.innerHTML = `
                <div class="col-checkbox" onclick="event.stopPropagation()">
                    <input type="checkbox" class="file-checkbox" data-key="${item.key}" ${selectedFiles.has(item.key) ? "checked" : ""}>
                </div>
                <div class="col-name">
                    <i class="${iconClass} item-icon"></i>
                    <span class="item-name" title="${item.name}">${item.name}</span>
                </div>
                <div class="col-size">${sizeStr}</div>
                <div class="col-date">${dateStr}</div>
                <div class="col-actions" onclick="event.stopPropagation()">
                    ${!item.is_dir ? `
                        <button class="btn-icon-sm" onclick="openPreview('${item.key}', '${item.name}')" title="Preview"><i class="fa-regular fa-eye"></i></button>
                        <button class="btn-icon-sm" onclick="openPresignedURLModal('${item.key}')" title="Create share link"><i class="fa-solid fa-share-nodes"></i></button>
                        <button class="btn-icon-sm" onclick="downloadFileDirectly('${item.key}', '${item.name}')" title="Download"><i class="fa-solid fa-download"></i></button>
                    ` : ""}
                    <button class="btn-icon-sm" onclick="openDeleteConfirmation(['${item.key}'])" title="Delete"><i class="fa-regular fa-trash-can"></i></button>
                </div>
            `;

            // Row click event for navigating folders
            row.onclick = () => {
                if (item.is_dir) {
                    currentPrefix = item.key;
                    clearSelection();
                    fetchFiles("");
                    updateURLAndHistory(true);
                } else {
                    // Toggle selection checkbox
                    const cb = row.querySelector(".file-checkbox");
                    cb.checked = !cb.checked;
                    if (cb.checked) {
                        row.classList.add("selected");
                        selectedFiles.add(item.key);
                    } else {
                        row.classList.remove("selected");
                        selectedFiles.delete(item.key);
                    }
                    updateBulkActionsBar();
                }
            };

            // Custom toggle handler for checkbox itself
            row.querySelector(".file-checkbox").onchange = (e) => {
                if (e.target.checked) {
                    row.classList.add("selected");
                    selectedFiles.add(item.key);
                } else {
                    row.classList.remove("selected");
                    selectedFiles.delete(item.key);
                }
                updateBulkActionsBar();
            };

            filesList.appendChild(row);
        });

    } catch (err) {
        loading.style.display = "none";
        showToast("error", err.message);
    }
}

// --- 6. Breadcrumbs ---
function renderBreadcrumbs(search = "") {
    const breadcrumb = document.getElementById("breadcrumb");
    breadcrumb.innerHTML = "";

    // Home item
    const home = document.createElement("span");
    home.className = `breadcrumb-item ${currentPrefix === "" && !search ? "active" : ""}`;
    home.innerHTML = `<i class="fa-solid fa-house"></i> Home`;
    home.onclick = () => {
        if (currentPrefix !== "" || search) {
            currentPrefix = "";
            document.getElementById("search-input").value = "";
            document.getElementById("btn-clear-search").style.display = "none";
            clearSelection();
            fetchFiles("");
            updateURLAndHistory(true);
        }
    };
    breadcrumb.appendChild(home);

    if (search) {
        const separator = document.createElement("span");
        separator.className = "breadcrumb-separator";
        separator.innerHTML = `<i class="fa-solid fa-chevron-right"></i>`;
        breadcrumb.appendChild(separator);

        const searchItem = document.createElement("span");
        searchItem.className = "breadcrumb-item active";
        searchItem.innerText = `Search: "${search}"`;
        breadcrumb.appendChild(searchItem);
        return;
    }

    if (currentPrefix === "") return;

    const parts = currentPrefix.split("/");
    let pathAcc = "";

    for (let i = 0; i < parts.length; i++) {
        const part = parts[i];
        if (part === "") continue;

        pathAcc += part + "/";

        const separator = document.createElement("span");
        separator.className = "breadcrumb-separator";
        separator.innerHTML = `<i class="fa-solid fa-chevron-right"></i>`;
        breadcrumb.appendChild(separator);

        const item = document.createElement("span");
        const isLast = (i === parts.length - 2 || (i === parts.length - 1 && !currentPrefix.endsWith("/")));
        item.className = `breadcrumb-item ${isLast ? "active" : ""}`;
        item.innerText = part;

        if (!isLast) {
            const currentPath = pathAcc;
            item.onclick = () => {
                currentPrefix = currentPath;
                clearSelection();
                fetchFiles("");
                updateURLAndHistory(true);
            };
        }
        breadcrumb.appendChild(item);
    }
}

// --- 7. Selection & Bulk Actions ---
function clearSelection() {
    selectedFiles.clear();
    const checkboxes = document.querySelectorAll(".file-checkbox");
    checkboxes.forEach(cb => cb.checked = false);
    const rows = document.querySelectorAll(".file-row");
    rows.forEach(row => row.classList.remove("selected"));
    document.getElementById("select-all-checkbox").checked = false;
    updateBulkActionsBar();
}

function updateBulkActionsBar() {
    const bar = document.getElementById("bulk-actions-bar");
    const countLabel = document.getElementById("selected-count");

    if (selectedFiles.size > 0) {
        bar.style.display = "flex";
        countLabel.innerText = `Selected ${selectedFiles.size} files`;
    } else {
        bar.style.display = "none";
    }
}

// --- 8. File Uploads (Presigned PUT URLs) ---
async function handleFileUploads(files) {
    if (files.length === 0) return;

    const pane = document.getElementById("upload-progress-pane");
    const list = document.getElementById("upload-items-list");
    const countLabel = document.getElementById("upload-active-count");

    pane.style.display = "flex";
    
    // Count active items
    let activeCount = parseInt(countLabel.innerText) || 0;
    activeCount += files.length;
    countLabel.innerText = activeCount;

    for (let i = 0; i < files.length; i++) {
        const file = files[i];
        
        // Target key structure: currentPrefix + filename
        const fileKey = currentPrefix + file.name;

        // 1. Create a progress bar item in progress pane
        const itemId = "upload-" + Math.random().toString(36).substr(2, 9);
        const itemElement = document.createElement("div");
        itemElement.className = "upload-item";
        itemElement.id = itemId;
        itemElement.innerHTML = `
            <div class="upload-item-info">
                <span class="upload-item-name" title="${file.name}">${file.name}</span>
                <span class="upload-item-percent" id="${itemId}-percent">0%</span>
            </div>
            <div class="progress-bar-container">
                <div class="progress-bar-fill" id="${itemId}-fill"></div>
            </div>
        `;
        list.appendChild(itemElement);

        // Perform async upload
        uploadSingleFile(file, fileKey, itemId, countLabel);
    }
}

async function uploadSingleFile(file, fileKey, itemId, countLabel) {
    const fill = document.getElementById(`${itemId}-fill`);
    const percentLabel = document.getElementById(`${itemId}-percent`);

    try {
        // 1. Ask Backend to generate S3 Presigned PUT URL for upload
        const presignRes = await fetch(`/api/connections/${activeConnection}/presigned-url`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                key: fileKey,
                action: "upload",
                expires: 30 // Expiry 30 minutes
            })
        });

        const presignData = await presignRes.json();
        if (!presignRes.ok) {
            throw new Error(presignData.error || "Failed to generate presigned upload URL");
        }

        // 2. Perform direct browser-to-S3 upload via XMLHttpRequest to monitor real progress bytes
        const xhr = new XMLHttpRequest();
        activeUploads[itemId] = xhr;

        xhr.open("PUT", presignData.url, true);
        
        // Crucial: Set Content-Type correctly to empty string or S3 matches signature
        xhr.setRequestHeader("Content-Type", file.type || "application/octet-stream");

        // Monitor real-time upload progress percentage
        xhr.upload.onprogress = (e) => {
            if (e.lengthComputable) {
                const percent = Math.round((e.loaded / e.total) * 100);
                fill.style.width = percent + "%";
                percentLabel.innerText = percent + "%";
            }
        };

        xhr.onload = () => {
            if (xhr.status === 200) {
                // Success!
                fill.className = "progress-bar-fill success";
                percentLabel.innerText = "Success";
                showToast("success", `Uploaded file: ${file.name}`);
                
                // Refresh folder list automatically
                fetchFiles("");
            } else {
                fill.className = "progress-bar-fill error";
                percentLabel.innerText = "S3 Error";
                showToast("error", `Upload failed for ${file.name} (S3 Status: ${xhr.status})`);
            }
            finalizeUploadItem(itemId, countLabel);
        };

        xhr.onerror = () => {
            fill.className = "progress-bar-fill error";
            percentLabel.innerText = "Network Error";
            showToast("error", `Network error uploading ${file.name}`);
            finalizeUploadItem(itemId, countLabel);
        };

        // Send binary data directly
        xhr.send(file);

    } catch (err) {
        fill.className = "progress-bar-fill error";
        percentLabel.innerText = "Failed";
        showToast("error", err.message);
        finalizeUploadItem(itemId, countLabel);
    }
}

function finalizeUploadItem(itemId, countLabel) {
    delete activeUploads[itemId];
    
    // Decrement active upload counter
    let activeCount = parseInt(countLabel.innerText) || 0;
    if (activeCount > 0) activeCount--;
    countLabel.innerText = activeCount;

    // Automatically remove success upload indicators after 5 seconds to prevent list clutter
    setTimeout(() => {
        const item = document.getElementById(itemId);
        if (item) item.remove();
        
        // Hide pane completely if no active uploads left
        if (Object.keys(activeUploads).length === 0) {
            document.getElementById("upload-progress-pane").style.display = "none";
        }
    }, 5000);
}

// --- 9. File Direct Download (Presigned GET URLs) ---
async function downloadFileDirectly(key, filename) {
    try {
        const res = await fetch(`/api/connections/${activeConnection}/presigned-url`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                key: key,
                action: "download",
                expires: 10 // Expire in 10 minutes
            })
        });

        const data = await res.json();
        if (!res.ok) throw new Error(data.error || "Failed to generate download link");

        // Force browser file download by opening presigned link in a temporary hidden anchor
        const a = document.createElement("a");
        a.href = data.url;
        a.download = filename;
        a.style.display = "none";
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        showToast("success", `Starting download: ${filename}`);
    } catch (err) {
        showToast("error", err.message);
    }
}

// --- 10. Create Folder Logic ---
async function handleCreateFolder(e) {
    e.preventDefault();
    const folderName = document.getElementById("folder-name").value.trim();

    if (!folderName) return;

    // Check for trailing slash to define folder key in S3
    const folderKey = currentPrefix + folderName + "/";

    try {
        // To create a folder placeholder in S3, we generate a presigned PUT URL and upload a 0-byte dummy payload!
        const presignRes = await fetch(`/api/connections/${activeConnection}/presigned-url`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                key: folderKey,
                action: "upload",
                expires: 15
            })
        });

        const presignData = await presignRes.json();
        if (!presignRes.ok) throw new Error(presignData.error || "Failed to generate presigned URL");

        // Perform S3 upload with empty body
        const s3Res = await fetch(presignData.url, {
            method: "PUT",
            headers: { "Content-Type": "application/octet-stream" },
            body: new Blob([])
        });

        if (!s3Res.ok) throw new Error(`S3 error: ${s3Res.status}`);

        showToast("success", `Folder "${folderName}" created successfully!`);
        closeModal("modal-folder");
        document.getElementById("form-folder").reset();
        fetchFiles("");
    } catch (err) {
        showToast("error", `Failed to create folder: ${err.message}`);
    }
}

// --- 11. Presigned URL Generator Modal ---
function openPresignedURLModal(key) {
    document.getElementById("presigned-file-key").value = key;
    document.getElementById("presigned-result-box").style.display = "none";
    openModal("modal-presigned");

    // Dynamic submit callback
    const btnSubmit = document.getElementById("btn-generate-presigned-submit");
    btnSubmit.onclick = async () => {
        const expiresMinutes = parseInt(document.getElementById("presigned-expires").value);
        
        try {
            const res = await fetch(`/api/connections/${activeConnection}/presigned-url`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    key: key,
                    action: "download",
                    expires: expiresMinutes
                })
            });

            const data = await res.json();
            if (!res.ok) throw new Error(data.error || "Failed to generate link");

            const resultBox = document.getElementById("presigned-result-box");
            const urlOutput = document.getElementById("presigned-url-output");

            urlOutput.value = data.url;
            resultBox.style.display = "block";
            showToast("success", "Presigned share link generated!");
        } catch (err) {
            showToast("error", err.message);
        }
    };
}

// --- 12. Delete Files (Batch & Single) ---
let pendingDeletes = [];

function openDeleteConfirmation(keys) {
    if (keys.length === 0) return;
    
    pendingDeletes = keys;
    
    const count = keys.length;
    const textLabel = document.getElementById("delete-confirm-text");
    
    if (count === 1) {
        // Extract plain filename
        const parts = keys[0].split("/");
        const name = parts[parts.length-1] || parts[parts.length-2] || keys[0];
        textLabel.innerText = `Are you sure you want to delete "${name}"? This action cannot be undone.`;
    } else {
        textLabel.innerText = `Are you sure you want to delete ${count} selected files? This action cannot be undone.`;
    }

    openModal("modal-confirm-delete");

    const submitBtn = document.getElementById("btn-confirm-delete-submit");
    submitBtn.onclick = handleExecuteDeletes;
}

async function handleExecuteDeletes() {
    if (pendingDeletes.length === 0) return;

    try {
        const res = await fetch(`/api/connections/${activeConnection}/files`, {
            method: "DELETE",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ keys: pendingDeletes })
        });

        const data = await res.json();
        if (!res.ok) throw new Error(data.error || "Failed to delete files");

        showToast("success", `Successfully deleted ${pendingDeletes.length} files!`);
        closeModal("modal-confirm-delete");
        clearSelection();
        fetchFiles("");
    } catch (err) {
        showToast("error", err.message);
    } finally {
        pendingDeletes = [];
    }
}

// --- 13. File Previews Logic ---
async function openPreview(key, filename) {
    const previewBody = document.getElementById("preview-body");
    const title = document.getElementById("preview-file-name");
    const downloadBtn = document.getElementById("btn-download-preview");

    title.innerText = `Preview: ${filename}`;
    downloadBtn.onclick = () => downloadFileDirectly(key, filename);

    // Initial Loading State
    previewBody.innerHTML = `
        <div class="viewport-loading">
            <i class="fa-solid fa-circle-notch spinner animate-spin"></i>
            <p>Loading file data...</p>
        </div>
    `;

    openModal("modal-preview");

    const ext = filename.split(".").pop().toLowerCase();

    // 1. If it's an Image format: generate presigned GET URL and display inside <img>
    if (["jpg", "jpeg", "png", "gif", "webp", "svg"].includes(ext)) {
        try {
            const url = await fetchPresignedUrl(key);
            previewBody.innerHTML = `
                <div class="preview-image-container">
                    <img src="${url}" alt="${filename}">
                </div>
            `;
        } catch (err) {
            previewBody.innerHTML = `<div class="connections-empty" style="color:var(--color-danger)"><i class="fa-solid fa-circle-exclamation"></i> Error: ${err.message}</div>`;
        }
        return;
    }

    // 2. If it's a PDF file: generate presigned GET URL and embed inside an <iframe>
    if (ext === "pdf") {
        try {
            const url = await fetchPresignedUrl(key);
            previewBody.innerHTML = `
                <div class="preview-iframe-container">
                    <iframe src="${url}"></iframe>
                </div>
            `;
        } catch (err) {
            previewBody.innerHTML = `<div class="connections-empty" style="color:var(--color-danger)"><i class="fa-solid fa-circle-exclamation"></i> Error: ${err.message}</div>`;
        }
        return;
    }

    // 3. For Text, JSON, CSV, Markdown: fetch content from backend reader
    if (["txt", "json", "csv", "md", "xml", "js", "go", "yaml", "yml"].includes(ext)) {
        try {
            const res = await fetch(`/api/connections/${activeConnection}/preview?key=${encodeURIComponent(key)}`);
            const data = await res.json();
            if (!res.ok) throw new Error(data.error || "Failed to read file content");

            let renderHtml = "";
            if (ext === "md") {
                // Markdown visual rendering wrapper
                renderHtml = `<div style="line-height:1.6; font-size:14px; color:var(--text-primary); max-width:800px; margin:0 auto; padding:16px;">
                    ${escapeHtml(data.content).replace(/\n/g, "<br>")}
                </div>`;
            } else if (ext === "json") {
                // Beautify JSON output
                try {
                    const parsed = JSON.parse(data.content);
                    const pretty = JSON.stringify(parsed, null, 4);
                    renderHtml = `<pre><code>${escapeHtml(pretty)}</code></pre>`;
                } catch {
                    renderHtml = `<pre><code>${escapeHtml(data.content)}</code></pre>`;
                }
            } else {
                renderHtml = `<pre><code>${escapeHtml(data.content)}</code></pre>`;
            }

            previewBody.innerHTML = renderHtml;
        } catch (err) {
            previewBody.innerHTML = `<div class="connections-empty" style="color:var(--color-danger)"><i class="fa-solid fa-circle-exclamation"></i> Error: ${err.message}</div>`;
        }
        return;
    }

    // 4. Unsupported preview formats
    previewBody.innerHTML = `
        <div class="viewport-placeholder">
            <i class="fa-regular fa-eye-slash placeholder-icon"></i>
            <h3>Preview not supported for this format</h3>
            <p>Previewing ".${ext}" files is not supported. Please click download to check the file.</p>
            <button class="btn btn-secondary" onclick="downloadFileDirectly('${key}', '${filename}')">
                <i class="fa-solid fa-download"></i> Download File
            </button>
        </div>
    `;
}

// Fetch presigned download URL for images and PDF previews helper
async function fetchPresignedUrl(key) {
    const res = await fetch(`/api/connections/${activeConnection}/presigned-url`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
            key: key,
            action: "download",
            expires: 10
        })
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || "Failed to generate link");
    return data.url;
}

// --- 14. Modals Management Helpers ---
function openModal(id) {
    document.getElementById(id).classList.add("active");
}

function closeModal(id) {
    document.getElementById(id).classList.remove("active");
    if (id === "modal-connection") {
        document.getElementById("conn-verify-status").innerHTML = "";
    }
}

// Close modals on clicking backdrop background
window.onclick = function(event) {
    if (event.target.classList.contains("modal")) {
        event.target.classList.remove("active");
        if (event.target.id === "modal-connection") {
            document.getElementById("conn-verify-status").innerHTML = "";
        }
    }
};

// --- 15. Utility Helpers ---
function formatBytes(bytes, decimals = 2) {
    if (bytes === 0) return "0 Bytes";
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ["Bytes", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
}

function getFileIcon(filename) {
    const ext = filename.split(".").pop().toLowerCase();
    switch (ext) {
        case "pdf": return "fa-regular fa-file-pdf";
        case "doc":
        case "docx": return "fa-regular fa-file-word";
        case "xls":
        case "xlsx": return "fa-regular fa-file-excel";
        case "ppt":
        case "pptx": return "fa-regular fa-file-powerpoint";
        case "png":
        case "jpg":
        case "jpeg":
        case "gif":
        case "webp":
        case "svg": return "fa-regular fa-file-image";
        case "zip":
        case "rar":
        case "tar":
        case "gz": return "fa-regular fa-file-zipper";
        case "mp3":
        case "wav":
        case "ogg": return "fa-regular fa-file-audio";
        case "mp4":
        case "mov":
        case "avi": return "fa-regular fa-file-video";
        case "txt": return "fa-regular fa-file-lines";
        case "json":
        case "js":
        case "html":
        case "css":
        case "go": return "fa-regular fa-file-code";
        default: return "fa-regular fa-file";
    }
}

function escapeHtml(unsafe) {
    return unsafe
         .replace(/&/g, "&amp;")
         .replace(/</g, "&lt;")
         .replace(/>/g, "&gt;")
         .replace(/"/g, "&quot;")
         .replace(/'/g, "&#039;");
}

// Toast notification popup system
function showToast(type, message) {
    const container = document.getElementById("toast-container");
    const toast = document.createElement("div");
    toast.className = `toast ${type}`;
    
    let iconClass = "fa-solid fa-circle-info";
    if (type === "success") iconClass = "fa-solid fa-circle-check";
    if (type === "error") iconClass = "fa-solid fa-circle-xmark";

    toast.innerHTML = `
        <i class="${iconClass}"></i>
        <span>${message}</span>
    `;

    container.appendChild(toast);

    // Auto fadeout after 4 seconds
    setTimeout(() => {
        toast.style.animation = "slideIn 0.3s cubic-bezier(0.18, 0.89, 0.32, 1.28) reverse forwards";
        setTimeout(() => toast.remove(), 300);
    }, 4000);
}

// --- 16. URL & History Management ---
function updateURLAndHistory(push = true) {
    let url = "/";
    if (activeConnection) {
        url += `?connection=${activeConnection}`;
        if (currentPrefix) {
            url += `&prefix=${encodeURIComponent(currentPrefix)}`;
        }
    }
    if (push) {
        history.pushState({ connectionId: activeConnection, prefix: currentPrefix }, "", url);
    }
}

function restoreStateFromURL() {
    const params = new URLSearchParams(window.location.search);
    const connId = params.get("connection");
    const prefix = params.get("prefix") || "";
    
    if (connId && connections.some(c => c.id === connId)) {
        activeConnection = connId;
        currentPrefix = prefix;
        
        // Refresh connection list formatting
        renderConnections();
        
        const activeConn = connections.find(c => c.id === connId);
        if (activeConn) {
            document.getElementById("active-connection-title").innerText = activeConn.name;
            document.getElementById("active-bucket-name").innerText = activeConn.bucket || "All Buckets";
            
            // Adjust S3 CRUD buttons based on bucket bounds
            const activeConnInList = connections.find(c => c.id === activeConnection);
            const showBtns = activeConnInList && (activeConnInList.bucket !== "" || prefix !== "");
            document.getElementById("btn-upload").style.display = showBtns ? "inline-flex" : "none";
            document.getElementById("btn-create-folder").style.display = showBtns ? "inline-flex" : "none";
            document.getElementById("nav-bar").style.display = "flex";
            document.getElementById("active-bucket-badge").style.display = activeConn.bucket ? "flex" : "none";
            document.getElementById("placeholder-select-conn").style.display = "none";
            
            clearSelection();
            fetchFiles("");
        }
    }
}
