import numpy as np
import plotly.graph_objects as go
from plotly.subplots import make_subplots
import plotly.express as px
from scipy.integrate import solve_ivp
from ipywidgets import interact, interactive, fixed, widgets
from IPython.display import display
import math

# Constants
MU_0 = 4 * np.pi * 1e-7  # Vacuum permeability (H/m)
MATERIALS = {
    "Iron": {"density": 7874, "relative_permeability": 5000},
    "304 Steel": {"density": 8000, "relative_permeability": 1.02},
    "Mild Steel": {"density": 7850, "relative_permeability": 100}
}

class ElectromagneticSimulation:
    def __init__(self):
        # Coil parameters
        self.coil_length = 20e-3  # m
        self.coil_OD = 30e-3  # m
        self.coil_ID = 4e-3  # m
        self.peak_current = 200  # A
        
        # Cylinder parameters
        self.cylinder_length = 20e-3  # m
        self.cylinder_diameter = 3e-3  # m
        self.material = "Iron"
        
        # Simulation parameters
        self.initial_position = -self.cylinder_length  # m (just touching the coil)
        self.initial_velocity = 0  # m/s
        self.friction_coefficient = 0.1  # unitless
        self.time_step = 1e-6  # s (1µs)
        self.max_time = 0.1  # s
        
        # Pulse parameters
        self.pulse_width = 1e-3  # s (1ms)
        self.pulse_skewness = 0.5  # unitless (0.5 means symmetric)
        
        # Results
        self.times = None
        self.positions = None
        self.velocities = None
        self.forces = None
        self.currents = None
        self.magnetic_energies = None
        self.kinetic_energies = None
        
        # Create figure
        self.create_figure()
        
    def create_figure(self):
        self.fig = make_subplots(
            rows=3, cols=2,
            subplot_titles=(
                "Simulation View", "Current Pulse",
                "Position vs Time", "Velocity vs Time",
                "Force vs Time", "Energy vs Time"
            ),
            specs=[
                [{"type": "scatter"}, {"type": "scatter"}],
                [{"type": "scatter"}, {"type": "scatter"}],
                [{"type": "scatter"}, {"type": "scatter"}]
            ],
            vertical_spacing=0.1,
            horizontal_spacing=0.1
        )
        
        # Add simulation view (static components)
        # Coil representation
        coil_x = np.linspace(0, self.coil_length, 100)
        coil_top_y = np.ones_like(coil_x) * self.coil_OD/2
        coil_bottom_y = np.ones_like(coil_x) * self.coil_ID/2
        
        self.fig.add_trace(
            go.Scatter(
                x=coil_x, y=coil_top_y,
                mode='lines', name='Coil (outer)',
                line=dict(color='darkorange', width=2),
            ),
            row=1, col=1
        )
        
        self.fig.add_trace(
            go.Scatter(
                x=coil_x, y=coil_bottom_y,
                mode='lines', name='Coil (inner)',
                line=dict(color='darkorange', width=2),
                fill='tonexty', fillcolor='rgba(255, 165, 0, 0.3)'
            ),
            row=1, col=1
        )
        
        self.fig.add_trace(
            go.Scatter(
                x=coil_x, y=-coil_top_y,
                mode='lines', showlegend=False,
                line=dict(color='darkorange', width=2),
            ),
            row=1, col=1
        )
        
        self.fig.add_trace(
            go.Scatter(
                x=coil_x, y=-coil_bottom_y,
                mode='lines', showlegend=False,
                line=dict(color='darkorange', width=2),
                fill='tonexty', fillcolor='rgba(255, 165, 0, 0.3)'
            ),
            row=1, col=1
        )
        
        # Dummy cylinder (will be updated)
        self.cylinder_trace = go.Scatter(
            x=[self.initial_position, self.initial_position + self.cylinder_length],
            y=[0, 0],
            mode='lines',
            name='Cylinder',
            line=dict(color='gray', width=self.cylinder_diameter * 1000)  # Scale for visibility
        )
        self.fig.add_trace(self.cylinder_trace, row=1, col=1)
        
        # Add current pulse preview
        self.pulse_preview_trace = go.Scatter(
            x=[], y=[],
            mode='lines',
            name='Current Pulse',
            line=dict(color='blue', width=2)
        )
        self.fig.add_trace(self.pulse_preview_trace, row=1, col=2)
        
        # Add placeholder traces for dynamic results
        self.position_trace = go.Scatter(x=[], y=[], mode='lines', name='Position', line=dict(color='green', width=2))
        self.velocity_trace = go.Scatter(x=[], y=[], mode='lines', name='Velocity', line=dict(color='blue', width=2))
        self.force_trace = go.Scatter(x=[], y=[], mode='lines', name='Force', line=dict(color='red', width=2))
        self.current_trace = go.Scatter(x=[], y=[], mode='lines', name='Current', line=dict(color='purple', width=2))
        self.magnetic_energy_trace = go.Scatter(x=[], y=[], mode='lines', name='Magnetic Energy', line=dict(color='orange', width=2))
        self.kinetic_energy_trace = go.Scatter(x=[], y=[], mode='lines', name='Kinetic Energy', line=dict(color='teal', width=2))
        
        self.fig.add_trace(self.position_trace, row=2, col=1)
        self.fig.add_trace(self.velocity_trace, row=2, col=2)
        self.fig.add_trace(self.force_trace, row=3, col=1)
        
        # Add both energy traces to the same plot
        self.fig.add_trace(self.magnetic_energy_trace, row=3, col=2)
        self.fig.add_trace(self.kinetic_energy_trace, row=3, col=2)
        self.fig.add_trace(self.current_trace, row=1, col=2)
        
        # Update layout
        self.fig.update_layout(
            height=800,
            width=1000,
            title_text="Electromagnetic Cylinder Simulation",
            showlegend=True,
        )
        
        # Set axis labels and ranges
        self.fig.update_xaxes(title_text="Position (m)", range=[-0.05, 0.05], row=1, col=1)
        self.fig.update_yaxes(title_text="Radius (m)", range=[-0.02, 0.02], row=1, col=1)
        
        self.fig.update_xaxes(title_text="Time (ms)", row=1, col=2)
        self.fig.update_yaxes(title_text="Current (A)", row=1, col=2)
        
        self.fig.update_xaxes(title_text="Time (ms)", row=2, col=1)
        self.fig.update_yaxes(title_text="Position (m)", row=2, col=1)
        
        self.fig.update_xaxes(title_text="Time (ms)", row=2, col=2)
        self.fig.update_yaxes(title_text="Velocity (m/s)", row=2, col=2)
        
        self.fig.update_xaxes(title_text="Time (ms)", row=3, col=1)
        self.fig.update_yaxes(title_text="Force (N)", row=3, col=1)
        
        self.fig.update_xaxes(title_text="Time (ms)", row=3, col=2)
        self.fig.update_yaxes(title_text="Energy (J)", row=3, col=2)
    
    def current_pulse(self, t, width, skewness, peak):
        """
        Generate a skewed bell curve for current.
        
        Args:
            t: Time (s)
            width: Pulse width (s)
            skewness: Skewness parameter (0-1), 0.5 is symmetric
            peak: Peak current (A)
        
        Returns:
            Current value at time t
        """
        # Adjusted time for skewness
        t_adj = t / width
        
        # Use skewed normal-like function
        if skewness == 0.5:  # Symmetric case
            return peak * np.exp(-(t_adj**2) / (2 * 0.1**2))
        else:
            # Asymmetric case
            mu = 0.5  # Center of pulse
            sigma = 0.2  # Width parameter
            
            if t_adj <= mu:
                # Left side - scale by skewness
                factor = (1 - skewness) * 2
                return peak * np.exp(-((t_adj - mu) ** 2) / (2 * (sigma * factor) ** 2))
            else:
                # Right side - scale by (1-skewness)
                factor = skewness * 2
                return peak * np.exp(-((t_adj - mu) ** 2) / (2 * (sigma * factor) ** 2))
    
    def update_pulse_preview(self):
        """Update the current pulse preview based on current parameters"""
        t = np.linspace(0, self.pulse_width * 5, 500)  # Show 5x pulse width
        current = np.array([self.current_pulse(ti, self.pulse_width, self.pulse_skewness, self.peak_current) for ti in t])
        
        # Update the preview trace
        self.pulse_preview_trace.x = t * 1000  # Convert to ms
        self.pulse_preview_trace.y = current
        
        # Update layout with appropriate range
        max_t = max(5, self.pulse_width * 5 * 1000)  # Convert to ms, at least 5ms
        self.fig.update_xaxes(range=[0, max_t], row=1, col=2)
        self.fig.update_yaxes(range=[0, self.peak_current * 1.1], row=1, col=2)
    
    def magnetic_field_at_center(self, current, position):
        """
        Calculate the magnetic field at the center of the coil.
        
        Args:
            current: Current in the coil (A)
            position: Position of the cylinder center relative to coil start (m)
        
        Returns:
            Magnetic field (T)
        """
        # Approximate number of turns based on dimensions
        wire_diameter = 1e-3  # Assume 1mm wire
        n_turns_radial = (self.coil_OD - self.coil_ID) / (2 * wire_diameter)
        n_turns_axial = self.coil_length / wire_diameter
        n_turns = n_turns_radial * n_turns_axial
        
        # Calculate position relative to coil center
        coil_center = self.coil_length / 2
        position_from_center = position - coil_center
        
        # Calculate magnetic field at center (simplified solenoid formula)
        coil_radius = (self.coil_ID + self.coil_OD) / 4  # Average radius
        
        # For positions inside or close to the coil
        if 0 <= position <= self.coil_length:
            B = MU_0 * n_turns * current / self.coil_length
        else:
            # Field drops off outside the coil (simplified approximation)
            distance_from_coil_center = abs(position_from_center)
            # Rapid drop-off with distance
            B = MU_0 * n_turns * current / self.coil_length * (coil_radius**2 / (coil_radius**2 + distance_from_coil_center**2))**1.5
            
        return B
    
    def magnetic_force(self, current, position, material="Iron"):
        """
        Calculate the magnetic force on the cylinder.
        
        Args:
            current: Current in the coil (A)
            position: Position of the cylinder center (m)
            material: Material of the cylinder
        
        Returns:
            Force (N)
        """
        # Calculate relative position of cylinder center to coil start
        cylinder_center = position + self.cylinder_length / 2
        position_rel = cylinder_center
        
        # Get the field at this position
        B = self.magnetic_field_at_center(current, position_rel)
        
        # Calculate magnetic force (simplified model)
        # F = gradient of (m·B) where m is magnetic moment
        # For ferromagnetic materials, we'll use a simplified approach
        
        # Volume of the cylinder
        volume = np.pi * (self.cylinder_diameter/2)**2 * self.cylinder_length
        
        # Calculate field gradient (simplified)
        delta = 1e-6  # Small distance for gradient calculation
        B1 = self.magnetic_field_at_center(current, position_rel - delta/2)
        B2 = self.magnetic_field_at_center(current, position_rel + delta/2)
        dBdx = (B2 - B1) / delta
        
        # Magnetic susceptibility (simplified)
        mu_r = MATERIALS[material]["relative_permeability"]
        chi = mu_r - 1  # Magnetic susceptibility
        
        # Force calculation (simplified for ferromagnetic materials)
        force = volume * chi * B * dBdx / MU_0
        
        return force

    def magnetic_energy(self, current, position, material="Iron"):
        """
        Calculate the energy stored in the magnetic field.
        
        Args:
            current: Current in the coil (A)
            position: Position of the cylinder center (m)
            material: Material of the cylinder
        
        Returns:
            Energy (J)
        """
        # Calculate position of cylinder center relative to coil start
        cylinder_center = position + self.cylinder_length / 2
        position_rel = cylinder_center
        
        # Get the field at this position
        B = self.magnetic_field_at_center(current, position_rel)
        
        # Volume of the coil
        coil_volume = np.pi * ((self.coil_OD/2)**2 - (self.coil_ID/2)**2) * self.coil_length
        
        # Volume of the cylinder
        cylinder_volume = np.pi * (self.cylinder_diameter/2)**2 * self.cylinder_length
        
        # Magnetic energy density in free space: u = B²/(2*μ₀)
        energy_density_free_space = B**2 / (2 * MU_0)
        
        # Magnetic energy density in material: u = B²/(2*μ₀*μᵣ)
        mu_r = MATERIALS[material]["relative_permeability"]
        energy_density_material = B**2 / (2 * MU_0 * mu_r)
        
        # Total energy (simplified)
        energy = energy_density_free_space * coil_volume + energy_density_material * cylinder_volume
        
        return energy
    
    def cylinder_mass(self, material="Iron"):
        """Calculate the mass of the cylinder"""
        volume = np.pi * (self.cylinder_diameter/2)**2 * self.cylinder_length
        density = MATERIALS[material]["density"]
        return volume * density
    
    def dynamics(self, t, y, material, friction_coef):
        """
        ODE for cylinder dynamics.
        
        Args:
            t: Time (s)
            y: State [position, velocity]
            material: Material of the cylinder
            friction_coef: Friction coefficient
        
        Returns:
            Derivatives [velocity, acceleration]
        """
        position, velocity = y
        
        # Current at this time
        current = self.current_pulse(t, self.pulse_width, self.pulse_skewness, self.peak_current)
        
        # Magnetic force
        force = self.magnetic_force(current, position, material)
        
        # Friction force (opposing motion)
        if velocity != 0:
            direction = velocity / abs(velocity)
            friction_force = -friction_coef * direction
        else:
            friction_force = 0
        
        # Mass
        mass = self.cylinder_mass(material)
        
        # Acceleration
        acceleration = (force + friction_force) / mass
        
        return [velocity, acceleration]
    
    def check_exit_condition(self, t, y, material, friction_coef):
        """Event function that triggers when cylinder exits the coil"""
        position = y[0]
        # Check if the trailing edge of the cylinder has passed the end of the coil
        if position > self.coil_length:
            return 0
        return 1
    
    check_exit_condition.terminal = True  # Stop integration when event occurs
    
    def run_simulation(self):
        """Run the simulation and update results"""
        # Initial conditions
        y0 = [self.initial_position, self.initial_velocity]
        
        # Time span
        t_span = (0, self.max_time)
        
        # Solve ODE
        solution = solve_ivp(
            fun=lambda t, y: self.dynamics(t, y, self.material, self.friction_coefficient),
            t_span=t_span,
            y0=y0,
            method='RK45',
            events=self.check_exit_condition,
            args=(self.material, self.friction_coefficient),
            max_step=self.time_step
        )
        
        # Extract results
        self.times = solution.t
        self.positions = solution.y[0]
        self.velocities = solution.y[1]
        
        # Calculate derived quantities
        self.currents = np.array([
            self.current_pulse(t, self.pulse_width, self.pulse_skewness, self.peak_current)
            for t in self.times
        ])
        
        self.forces = np.array([
            self.magnetic_force(I, pos, self.material)
            for I, pos in zip(self.currents, self.positions)
        ])
        
        self.magnetic_energies = np.array([
            self.magnetic_energy(I, pos, self.material)
            for I, pos in zip(self.currents, self.positions)
        ])
        
        mass = self.cylinder_mass(self.material)
        self.kinetic_energies = 0.5 * mass * self.velocities**2
        
        # Update plots
        self.update_results_plots()
    
    def update_results_plots(self):
        """Update the plots with simulation results"""
        if self.times is None:
            return
        
        # Convert time to ms for plotting
        times_ms = self.times * 1000
        
        # Update position trace
        self.position_trace.x = times_ms
        self.position_trace.y = self.positions
        
        # Update velocity trace
        self.velocity_trace.x = times_ms
        self.velocity_trace.y = self.velocities
        
        # Update force trace
        self.force_trace.x = times_ms
        self.force_trace.y = self.forces
        
        # Update current trace
        self.current_trace.x = times_ms
        self.current_trace.y = self.currents
        
        # Update energy traces
        self.magnetic_energy_trace.x = times_ms
        self.magnetic_energy_trace.y = self.magnetic_energies
        
        self.kinetic_energy_trace.x = times_ms
        self.kinetic_energy_trace.y = self.kinetic_energies
        
        # Update cylinder position in visualization
        final_pos = self.positions[-1]
        self.update_cylinder_position(final_pos)
        
        # Update ranges for all plots
        max_time = max(times_ms)
        
        self.fig.update_xaxes(range=[0, max_time], row=2, col=1)
        self.fig.update_xaxes(range=[0, max_time], row=2, col=2)
        self.fig.update_xaxes(range=[0, max_time], row=3, col=1)
        self.fig.update_xaxes(range=[0, max_time], row=3, col=2)
        
        self.fig.update_yaxes(range=[min(self.positions) * 1.1, max(self.positions) * 1.1], row=2, col=1)
        self.fig.update_yaxes(range=[min(self.velocities) * 1.1, max(self.velocities) * 1.1], row=2, col=2)
        self.fig.update_yaxes(range=[min(self.forces) * 1.1, max(self.forces) * 1.1], row=3, col=1)
        
        max_energy = max(max(self.magnetic_energies), max(self.kinetic_energies)) * 1.1
        self.fig.update_yaxes(range=[0, max_energy], row=3, col=2)
    
    def update_cylinder_position(self, position):
        """Update the cylinder visualization with a new position"""
        self.cylinder_trace.x = [position, position + self.cylinder_length]
    
    def update_parameter(self, param_name, value):
        """Update a parameter and refresh relevant visualizations"""
        setattr(self, param_name, value)
        
        # Special updates for certain parameters
        if param_name == 'initial_position':
            self.update_cylinder_position(value)
        
        if param_name in ['pulse_width', 'pulse_skewness', 'peak_current']:
            self.update_pulse_preview()
        
        # Return the figure for display
        return self.fig
    
    def display_interactive(self):
        """Set up interactive widgets and display simulation"""
        # Update pulse preview initially
        self.update_pulse_preview()
        
        # Create widgets
        material_widget = widgets.Dropdown(
            options=list(MATERIALS.keys()),
            value=self.material,
            description='Material:',
            disabled=False
        )
        
        initial_position_widget = widgets.FloatSlider(
            value=self.initial_position,
            min=-self.cylinder_length*2,
            max=0,
            step=0.001,
            description='Initial Position (m):',
            disabled=False,
            continuous_update=True,
            orientation='horizontal',
            readout=True,
            readout_format='.3f',
        )
        
        initial_velocity_widget = widgets.FloatSlider(
            value=self.initial_velocity,
            min=0,
            max=100,
            step=1,
            description='Initial Velocity (m/s):',
            disabled=False,
            continuous_update=True,
            orientation='horizontal',
            readout=True,
            readout_format='.1f',
        )
        
        friction_coefficient_widget = widgets.FloatSlider(
            value=self.friction_coefficient,
            min=0,
            max=1,
            step=0.01,
            description='Friction Coefficient:',
            disabled=False,
            continuous_update=True,
            orientation='horizontal',
            readout=True,
            readout_format='.2f',
        )
        
        time_step_widget = widgets.FloatLogSlider(
            value=self.time_step,
            base=10,
            min=-9,  # 10^-9 = 1ns
            max=-6,  # 10^-6 = 1µs
            step=0.1,
            description='Time Step (s):',
            disabled=False,
            continuous_update=True,
            orientation='horizontal',
            readout=True,
            readout_format='.2e',
        )
        
        pulse_width_widget = widgets.FloatSlider(
            value=self.pulse_width * 1000,  # Convert to ms
            min=0.1,
            max=5,  # 5ms max
            step=0.1,
            description='Pulse Width (ms):',
            disabled=False,
            continuous_update=True,
            orientation='horizontal',
            readout=True,
            readout_format='.1f',
        )
        
        pulse_skewness_widget = widgets.FloatSlider(
            value=self.pulse_skewness,
            min=0,
            max=1,
            step=0.01,
            description='Pulse Skewness:',
            disabled=False,
            continuous_update=True,
            orientation='horizontal',
            readout=True,
            readout_format='.2f',
        )
        
        peak_current_widget = widgets.FloatSlider(
            value=self.peak_current,
            min=0,
            max=500,
            step=10,
            description='Peak Current (A):',
            disabled=False,
            continuous_update=True,
            orientation='horizontal',
            readout=True,
            readout_format='.0f',
        )
        
        run_button = widgets.Button(
            description='Run Simulation',
            disabled=False,
            button_style='success',
            tooltip='Click to run the simulation',
            icon='play'
        )
        
        # Set up callbacks
        def on_material_change(change):
            self.material = change['new']
        
        def on_initial_position_change(change):
            self.initial_position = change['new']
            self.update_cylinder_position(change['new'])
            display(self.fig)
        
        def on_initial_velocity_change(change):
            self.initial_velocity = change['new']
        
        def on_friction_coefficient_change(change):
            self.friction_coefficient = change['new']
        
        def on_time_step_change(change):
            self.time_step = change['new']
        
        def on_pulse_width_change(change):
            self.pulse_width = change['new'] / 1000  # Convert from ms to s
            self.update_pulse_preview()
            display(self.fig)
        
        def on_pulse_skewness_change(change):
            self.pulse_skewness = change['new']
            self.update_pulse_preview()
            display(self.fig)
        
        def on_peak_current_change(change):
            self.peak_current = change['new']
            self.update_pulse_preview()
            display(self.fig)
        
        def on_run_button_click(b):
            self.run_simulation()
            display(self.fig)
        
        # Register callbacks
        material_widget.observe(on_material_change, names='value')
        initial_position_widget.observe(on_initial_position_change, names='value')
        initial_velocity_widget.observe(on_initial_velocity_change, names='value')
        friction_coefficient_widget.observe(on_friction_coefficient_change, names='value')
        time_step_widget.observe(on_time_step_change, names='value')
        pulse_width_widget.observe(on_pulse_width_change, names='value')
        pulse_skewness_widget.observe(on_pulse_skewness_change, names='value')
        peak_current_widget.observe(on_peak_current_change, names='value')
        run_button.on_click(on_run_button_click)
        
        # Display widgets
        display(material_widget)
        display(initial_position_widget)
        display(initial_velocity_widget)
        display(friction_coefficient_widget)
        display(time_step_widget)
        display(pulse_width_widget)
        display(pulse_skewness_widget)
        display(peak_current_widget)
        display(run_button)
        
        # Display initial figure
        display(self.fig)

# Create and run the simulation
sim = ElectromagneticSimulation()
sim.display_interactive()