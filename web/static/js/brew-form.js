/**
 * Alpine.js component for the brew form
 * Manages pours, new entity modals, and form state
 */
function brewForm() {
    return {
        showNewBean: false,
        showNewGrinder: false,
        showNewBrewer: false,
        rating: 5,
        pours: [],
        newBean: { name: '', origin: '', roasterRKey: '', roastLevel: '', process: '', description: '' },
        newGrinder: { name: '', grinderType: '', burrType: '', notes: '' },
        newBrewer: { name: '', description: '' },
        
        init() {
            // Load existing pours if editing
            const poursData = this.$el.getAttribute('data-pours');
            if (poursData) {
                try {
                    this.pours = JSON.parse(poursData);
                } catch (e) {
                    console.error('Failed to parse pours data:', e);
                    this.pours = [];
                }
            }
        },
        
        addPour() {
            this.pours.push({ water: '', time: '' });
        },
        
        removePour(index) {
            this.pours.splice(index, 1);
        },
        
        async addBean() {
            if (!this.newBean.name || !this.newBean.origin) {
                alert('Bean name and origin are required');
                return;
            }
            const payload = {
                name: this.newBean.name,
                origin: this.newBean.origin,
                roast_level: this.newBean.roastLevel,
                process: this.newBean.process,
                description: this.newBean.description,
                roaster_rkey: this.newBean.roasterRKey || ''
            };
            const response = await fetch('/api/beans', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            if (response.ok) {
                window.location.reload();
            } else {
                const errorText = await response.text();
                alert('Failed to add bean: ' + errorText);
            }
        },
        
        async addGrinder() {
            if (!this.newGrinder.name) {
                alert('Grinder name is required');
                return;
            }
            const response = await fetch('/api/grinders', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(this.newGrinder)
            });
            if (response.ok) {
                window.location.reload();
            } else {
                const errorText = await response.text();
                alert('Failed to add grinder: ' + errorText);
            }
        },
        
        async addBrewer() {
            if (!this.newBrewer.name) {
                alert('Brewer name is required');
                return;
            }
            const response = await fetch('/api/brewers', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(this.newBrewer)
            });
            if (response.ok) {
                window.location.reload();
            } else {
                const errorText = await response.text();
                alert('Failed to add brewer: ' + errorText);
            }
        }
    }
}
