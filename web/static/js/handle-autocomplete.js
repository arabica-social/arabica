/**
 * Handle autocomplete for AT Protocol login
 * Provides typeahead search for Bluesky handles
 */
(function() {
    const input = document.getElementById('handle');
    const results = document.getElementById('autocomplete-results');
    
    // Exit early if elements don't exist (user might be authenticated)
    if (!input || !results) return;
    
    let debounceTimeout;
    let abortController;
    
    function debounce(func, wait) {
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(debounceTimeout);
                func(...args);
            };
            clearTimeout(debounceTimeout);
            debounceTimeout = setTimeout(later, wait);
        };
    }
    
    async function searchActors(query) {
        // Need at least 3 characters to search
        if (query.length < 3) {
            results.classList.add('hidden');
            results.innerHTML = '';
            return;
        }
        
        // Cancel previous request
        if (abortController) {
            abortController.abort();
        }
        abortController = new AbortController();
        
        try {
            const response = await fetch(`/api/search-actors?q=${encodeURIComponent(query)}`, {
                signal: abortController.signal
            });
            
            if (!response.ok) {
                results.classList.add('hidden');
                results.innerHTML = '';
                return;
            }
            
            const data = await response.json();
            
            if (!data.actors || data.actors.length === 0) {
                results.innerHTML = '<div class="px-4 py-3 text-sm text-gray-500">No accounts found</div>';
                results.classList.remove('hidden');
                return;
            }
            
            // Display the actors
            results.innerHTML = data.actors.map(actor => {
                const avatarUrl = actor.avatar || '/static/icon-placeholder.svg';
                const displayName = actor.displayName || actor.handle;
                
                return `
                    <div class="handle-result px-3 py-2 hover:bg-gray-100 cursor-pointer flex items-center gap-2"
                         data-handle="${actor.handle}">
                        <img src="${avatarUrl}" 
                             alt="${displayName}"
                             width="32"
                             height="32"
                             class="w-6 h-6 rounded-full object-cover flex-shrink-0"
                             onerror="this.src='/static/icon-placeholder.svg'" />
                        <div class="flex-1 min-w-0">
                            <div class="font-medium text-sm text-gray-900 truncate">${displayName}</div>
                            <div class="text-xs text-gray-500 truncate">@${actor.handle}</div>
                        </div>
                    </div>
                `;
            }).join('');
            
            results.classList.remove('hidden');
            
            // Add click handlers
            results.querySelectorAll('.handle-result').forEach(el => {
                el.addEventListener('click', function() {
                    input.value = this.dataset.handle;
                    results.classList.add('hidden');
                    results.innerHTML = '';
                });
            });
        } catch (error) {
            if (error.name !== 'AbortError') {
                console.error('Error searching actors:', error);
            }
        }
    }
    
    const debouncedSearch = debounce(searchActors, 300);
    
    input.addEventListener('input', function(e) {
        debouncedSearch(e.target.value);
    });
    
    // Hide results when clicking outside
    document.addEventListener('click', function(e) {
        if (!input.contains(e.target) && !results.contains(e.target)) {
            results.classList.add('hidden');
        }
    });
    
    // Show results again when input is focused
    input.addEventListener('focus', function() {
        if (results.innerHTML && input.value.length >= 3) {
            results.classList.remove('hidden');
        }
    });
})();
