class ImageProcessor {
    constructor() {
        this.apiUrl = '';
        this.currentPage = 1;
        this.limit = 12;
        this.currentImageId = null;
        this.autoRefreshInterval = null;

        this.initElements();
        this.attachEventListeners();
        this.loadImages();
        this.startAutoRefresh();
    }

    initElements() {
        // Upload form
        this.uploadForm = document.getElementById('uploadForm');
        this.imageFile = document.getElementById('imageFile');
        this.fileName = document.getElementById('fileName');
        this.uploadBtn = document.getElementById('uploadBtn');
        this.uploadProgress = document.getElementById('uploadProgress');

        // Images
        this.imagesContainer = document.getElementById('imagesContainer');
        this.loading = document.getElementById('loading');
        this.refreshBtn = document.getElementById('refreshBtn');

        // Pagination
        this.prevBtn = document.getElementById('prevBtn');
        this.nextBtn = document.getElementById('nextBtn');
        this.pageInfo = document.getElementById('pageInfo');

        // Modal
        this.imageModal = document.getElementById('imageModal');
        this.closeModal = document.querySelector('.close-modal');
        this.modalImage = document.getElementById('modalImage');
        this.modalTitle = document.getElementById('modalTitle');
        this.modalStatus = document.getElementById('modalStatus');
        this.modalSize = document.getElementById('modalSize');
        this.modalProcessing = document.getElementById('modalProcessing');
        this.modalDate = document.getElementById('modalDate');
        this.downloadOriginal = document.getElementById('downloadOriginal');
        this.downloadProcessed = document.getElementById('downloadProcessed');
        this.deleteBtn = document.getElementById('deleteBtn');
    }

    attachEventListeners() {
        // File input
        this.imageFile.addEventListener('change', (e) => {
            if (e.target.files.length > 0) {
                this.fileName.textContent = e.target.files[0].name;
            }
        });

        // Upload form
        this.uploadForm.addEventListener('submit', (e) => {
            e.preventDefault();
            this.uploadImage();
        });

        // Refresh
        this.refreshBtn.addEventListener('click', () => this.loadImages());

        // Pagination
        this.prevBtn.addEventListener('click', () => this.prevPage());
        this.nextBtn.addEventListener('click', () => this.nextPage());

        // Modal
        this.closeModal.addEventListener('click', () => this.closeImageModal());
        this.imageModal.addEventListener('click', (e) => {
            if (e.target === this.imageModal) this.closeImageModal();
        });
        this.deleteBtn.addEventListener('click', () => this.deleteImage());

        // Keyboard
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.imageModal.style.display === 'block') {
                this.closeImageModal();
            }
        });
    }

    async apiCall(url, options = {}) {
        try {
            const res = await fetch(url, options);
            if (!res.ok) {
                const err = await res.json().catch(() => ({ error: 'Unknown error' }));
                throw new Error(err.message || err.error || 'Request failed');
            }
            return res.status === 204 ? null : await res.json();
        } catch (e) {
            this.showError(e.message);
            throw e;
        }
    }

    async uploadImage() {
        const file = this.imageFile.files[0];
        if (!file) {
            this.showError('Выберите файл');
            return;
        }

        const formData = new FormData(this.uploadForm);

        this.uploadBtn.disabled = true;
        this.uploadProgress.style.display = 'block';

        try {
            const result = await this.apiCall('/upload', {
                method: 'POST',
                body: formData
            });

            this.showSuccess('Изображение загружено! Обработка началась...');
            this.uploadForm.reset();
            this.fileName.textContent = 'Выберите файл';
            this.loadImages();
        } catch (e) {
            console.error('Upload failed:', e);
        } finally {
            this.uploadBtn.disabled = false;
            this.uploadProgress.style.display = 'none';
        }
    }

    async loadImages() {
        this.loading.style.display = 'block';
        this.imagesContainer.innerHTML = '<div class="loading">Загрузка...</div>';

        const offset = (this.currentPage - 1) * this.limit;

        try {
            const data = await this.apiCall(`/images?limit=${this.limit}&offset=${offset}`);
            this.renderImages(data.images || []);
            this.updatePagination(data.images?.length === this.limit);
        } catch (e) {
            this.imagesContainer.innerHTML = '<div class="loading">Ошибка загрузки</div>';
        } finally {
            this.loading.style.display = 'none';
        }
    }

    renderImages(images) {
        this.imagesContainer.innerHTML = '';

        if (images.length === 0) {
            this.imagesContainer.innerHTML = '<div class="loading">Изображений пока нет</div>';
            return;
        }

        images.forEach(img => {
            const card = this.createImageCard(img);
            this.imagesContainer.appendChild(card);
        });
    }

    createImageCard(img) {
        const card = document.createElement('div');
        card.className = 'image-card';
        card.onclick = () => this.openImageModal(img);

        const thumbnail = document.createElement('img');
        thumbnail.className = 'image-thumbnail';

        // Показываем обработанное изображение или оригинал
        if (img.status === 'completed' && img.processed_url) {
            thumbnail.src = img.processed_url;
        } else {
            thumbnail.src = img.original_url;
        }
        thumbnail.alt = img.original_filename;
        thumbnail.onerror = () => {
            thumbnail.src = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="200" height="200"%3E%3Crect fill="%23ddd" width="200" height="200"/%3E%3Ctext x="50%25" y="50%25" text-anchor="middle" fill="%23999"%3ENo Image%3C/text%3E%3C/svg%3E';
        };

        const info = document.createElement('div');
        info.className = 'image-info';

        const filename = document.createElement('div');
        filename.className = 'image-filename';
        filename.textContent = img.original_filename;

        const meta = document.createElement('div');
        meta.className = 'image-meta';
        meta.innerHTML = `
            <div>${this.formatSize(img.size)}</div>
            <span class="status-badge status-${img.status}">${this.getStatusText(img.status)}</span>
        `;

        info.appendChild(filename);
        info.appendChild(meta);

        card.appendChild(thumbnail);
        card.appendChild(info);

        return card;
    }

    openImageModal(img) {
        this.currentImageId = img.id;

        // Показываем обработанное изображение если готово, иначе оригинал
        if (img.status === 'completed' && img.processed_url) {
            this.modalImage.src = img.processed_url;
        } else {
            this.modalImage.src = img.original_url;
        }

        this.modalTitle.textContent = img.original_filename;
        this.modalStatus.textContent = this.getStatusText(img.status);
        this.modalStatus.className = `status-badge status-${img.status}`;
        this.modalSize.textContent = this.formatSize(img.size);
        this.modalProcessing.textContent = this.getProcessingTypeText(img.processing_type);
        this.modalDate.textContent = new Date(img.created_at).toLocaleString();

        // Ссылки на скачивание
        this.downloadOriginal.href = img.original_url;
        this.downloadOriginal.download = img.original_filename;

        if (img.status === 'completed' && img.processed_url) {
            this.downloadProcessed.style.display = 'inline-block';
            this.downloadProcessed.href = img.processed_url;
            this.downloadProcessed.download = `processed_${img.original_filename}`;
        } else {
            this.downloadProcessed.style.display = 'none';
        }

        this.imageModal.style.display = 'block';
    }

    closeImageModal() {
        this.imageModal.style.display = 'none';
        this.currentImageId = null;
    }

    async deleteImage() {
        if (!this.currentImageId) return;

        if (!confirm('Удалить это изображение?')) return;

        try {
            await this.apiCall(`/image/${this.currentImageId}`, { method: 'DELETE' });
            this.showSuccess('Изображение удалено');
            this.closeImageModal();
            this.loadImages();
        } catch (e) {
            console.error('Delete failed:', e);
        }
    }

    updatePagination(hasMore) {
        this.pageInfo.textContent = `Страница ${this.currentPage}`;
        this.prevBtn.disabled = this.currentPage <= 1;
        this.nextBtn.disabled = !hasMore;
    }

    prevPage() {
        if (this.currentPage > 1) {
            this.currentPage--;
            this.loadImages();
        }
    }

    nextPage() {
        this.currentPage++;
        this.loadImages();
    }

    startAutoRefresh() {
        // Автоматически обновляем список каждые 5 секунд
        this.autoRefreshInterval = setInterval(() => {
            // Обновляем только если модальное окно закрыто
            if (this.imageModal.style.display !== 'block') {
                this.loadImages();
            }
        }, 5000);
    }

    // Utility methods
    formatSize(bytes) {
        if (bytes < 1024) return bytes + ' B';
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
        return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
    }

    getStatusText(status) {
        const statusMap = {
            'pending': '⏳ Ожидает',
            'processing': '⚙️ Обработка',
            'completed': '✅ Готово',
            'failed': '❌ Ошибка'
        };
        return statusMap[status] || status;
    }

    getProcessingTypeText(type) {
        const typeMap = {
            'resize': '🔄 Resize',
            'thumbnail': '🖼️ Thumbnail',
            'watermark': '💧 Watermark'
        };
        return typeMap[type] || type;
    }

    showSuccess(message) {
        alert(message); // Можно заменить на более красивое уведомление
    }

    showError(message) {
        alert('Ошибка: ' + message);
    }
}

// Инициализация при загрузке страницы
document.addEventListener('DOMContentLoaded', () => {
    new ImageProcessor();
}); 