# Import necessary libraries
import numpy as np
from scipy import integrate

from model import polynomial_curve_function, seasonal_curve_function


def compute_gaussian_distribution(x, y, mean, std_):
    """
    Evaluate the Gaussian distribution function with given mean and standard deviation.

    Parameters:
    - x (float): Value of x.
    - y (float): Value of y.
    - mean (function): Mean function.
    - std_ (float): Standard deviation.

    Returns:
    - float: Gaussian distribution value.
    """
    return np.exp(-(((y - mean(x)) ** 2) / (2 * (std_ ** 2))))


def generate_surface_function(std_, mean):
    """
    Generate a surface function using the Gaussian distribution.

    Parameters:
    - std_ (float): Standard deviation.
    - mean (function): Mean function.

    Returns:
    - function: Surface function.
    """

    # Define a surface function using the Gaussian distribution
    def surface_(y, x):
        return compute_gaussian_distribution(x, y, mean=mean, std_=std_)

    return surface_


def calculate_probability(x_l, x_u, y_l, y_u, surf, y_l_b):
    """
    Calculate the probability of y > y_lower_bound within the specified x range.

    Parameters:
    - x_l (float): Lower limit of x range.
    - x_u (float): Upper limit of x range.
    - y_l (float): Lower limit of y range.
    - y_u (float): Upper limit of y range.
    - surf (function): Surface function.
    - y_l_b (float): Lower bound of y.

    Returns:
    - float: Probability.
    """
    # Perform double integration to calculate segment volume and total volume
    vol_seg = integrate.dblquad(surf, x_l, x_u, y_l_b, y_u)
    print(f"Segment volume and error: {vol_seg}")
    vol_tot = integrate.dblquad(surf, x_l, x_u, y_l, y_u)
    print(f"Total volume and error: {vol_tot}")
    # Calculate the probability of y > y_lower_bound within the specified x range
    seg_frac = (vol_seg[0] / vol_tot[0]) * 100 if vol_tot[0] != 0 else 0
    res = round(seg_frac, 2)
    print(f"Probability of y>{y_l_b} for {x_l}<x<{x_u} estimated to {res}%")
    return res


def compute_out_of_bounds_probability(T, S, std, x_lower_limit, x_upper_limit, y_lower_bound):
    """
    Compute the probability of y being out of bounds for the specified trend.

    Parameters:
    - deserialized_trend (function): Deserialized trend function.
    - std (float): Standard deviation.
    - x_lower_limit (float): Lower limit of x.
    - x_upper_limit (float): Upper limit of x.
    - y_lower_bound (float): Lower bound of y.

    Returns:
    - float: Probability.
    - function: Surface function.
    - float: Lower limit of y.
    - float: Upper limit of y.
    """

    def fitted_function(x):
        return polynomial_curve_function(x, T[0], T[1], T[2], T[3]) + seasonal_curve_function(x, S[0], S[1], S[2], S[3])

    # Generate x values for the trend
    x_values = np.linspace(x_lower_limit, x_upper_limit, 1000)
    # Calculate the trend values
    trend_values = fitted_function(x_values)
    # Calculate the maximum and minimum trend values
    max_trend_value = np.max(trend_values)
    min_trend_value = np.min(trend_values)
    # Set the lower and upper limits of y based on trend values and standard deviation
    y_lower_limit = min_trend_value - 4 * std
    y_upper_limit = max_trend_value + 4 * std
    # Generate the surface function using the trend and standard deviation
    surface = generate_surface_function(std, fitted_function)
    # Calculate the probability of y being out of bounds
    prob = calculate_probability(x_lower_limit, x_upper_limit, y_lower_limit, y_upper_limit, surface, y_lower_bound)
    return prob, surface, y_lower_limit, y_upper_limit
