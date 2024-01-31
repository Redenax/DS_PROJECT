import numpy as np
import pandas as pd
from scipy.optimize import curve_fit
from scipy.stats import norm
from sklearn.linear_model import LinearRegression
from sklearn.metrics import mean_squared_error, r2_score
from sklearn.preprocessing import PolynomialFeatures
from statsmodels.tsa.filters.hp_filter import hpfilter
from statsmodels.tsa.seasonal import seasonal_decompose

"""
Overall, the code is a pipeline for analyzing time series data, decomposing it into components, fitting polynomial
and seasonal curves, and evaluating the quality of the fit. It provides insights into the seasonal and trend patterns
present in the data and how well they can be captured by the chosen models.
"""


def format_data(data):
    """
    Convert the provided list of tuples into a pandas DataFrame.

    Args:
    - data: List of tuples containing timestamp and corresponding values.

    Returns:
    - df: Pandas DataFrame with timestamp and value columns.
    """
    df = pd.DataFrame(data, columns=['timestamp_sec', 'value'])
    df['value'] = df['value'].astype(float)
    return df


def apply_hp_filter_with_optimal_lambda(time_series, lambda_range):
    """
    Apply the Hodrick-Prescott filter to the time series with optimal lambda in given range.

    Args:
    - time_series: Time series data.
    - lambda_range: List of lambda values to try.

    Returns:
    - best_cycle: Cycle component obtained from the filter.
    - best_trend: Trend component obtained from the filter.
    - best_lambda: Optimal lambda value.
    """
    best_lambda = None
    best_trend = None
    best_cycle = None
    best_variance_ratio = float('inf')

    # Iterate over lambda values to find the optimal one
    for lambda_ in lambda_range:
        # Apply Hodrick-Prescott filter
        cycle_, trend_ = hpfilter(time_series, lamb=lambda_)
        variance_ratio = np.var(cycle_) / np.var(time_series)
        # Update if variance ratio is improved
        if variance_ratio < best_variance_ratio:
            best_lambda = lambda_
            best_trend = trend_
            best_cycle = cycle_
            best_variance_ratio = variance_ratio

    return best_cycle, best_trend, best_lambda


def seasonal_decomposition(time_series, period_range):
    """
    Perform seasonal decomposition on the time series.

    Args:
    - time_series: Time series data.
    - period_range: List of periods to consider.

    Returns:
    - best_result: Seasonal decomposition result with optimal period.
    """
    best_period = None
    best_result = None
    best_seasonal_variance = float('-inf')
    best_residual_variance = float('inf')

    # Iterate over periods to find the optimal one
    for period in period_range:
        if time_series.size < 2 * period:
            break
        result_ = seasonal_decompose(time_series, model='additive', period=period)
        seasonal_variance = np.var(result_.seasonal)
        residual_variance = np.var(result_.resid)

        # Update if seasonal variance is higher and residual variance is lower
        if seasonal_variance > best_seasonal_variance and residual_variance < best_residual_variance:
            best_period = period
            best_result = result_
            best_seasonal_variance = seasonal_variance
            best_residual_variance = residual_variance

    print(f"Optimal period: {best_period}")
    return best_result


def polynomial_fit_and_plot(x, y, degree=3):
    """
    Fit a polynomial of specified degree to the data and return the coefficients.

    Args:
    - x: Input feature.
    - y: Target variable.
    - degree: Degree of the polynomial (default is 1).

    Returns:
    - coefficients: Coefficients of the fitted polynomial.
    """
    # Create polynomial features
    pr = PolynomialFeatures(degree=degree)
    x_poly = pr.fit_transform(x)
    # Fit linear regression
    lr_2 = LinearRegression()
    lr_2.fit(x_poly, y)
    coefficients = lr_2.coef_[0]
    print("Polynomial coefficients :", coefficients)
    if degree < 3:
        coefficients = np.concatenate((coefficients, np.zeros(3 - degree)))

    return coefficients


def seasonal_curve_function(x, a, b, c, d):
    """
    Define the function for seasonal curve fitting.

    Args:
    - x: Input variable.
    - a, b, c: Parameters for curve fitting.

    Returns:
    - Fitted values.
    """
    return a * np.sin(b * x + c) + d


def polynomial_curve_function(x, a, b, c, d):
    """
    Define the function for trend curve fitting.

    Args:
    - x: Input variable.
    - a, b, c: Parameters for curve fitting.

    Returns:
    - Fitted values.
    """
    return a + b * x + c * x ** 2 + d * x ** 3


def seasonal_curve_fit_and_plot(f, x, y):
    """
    Fit the seasonal curve to the data using curve_fit.

    Args:
    - f: Function for curve fitting.
    - x: Input variable.
    - y: Target variable.

    Returns:
    - Fitted parameters.
    """
    # Use curve_fit to fit the function to the data
    res = curve_fit(f, x, y)
    print("Curve fit coefficients :", res[0])
    return res[0]


def complete_fit_and_plot(x, y, T, S):
    """
    Combine polynomial and seasonal curve fits to create a complete fit.

    Args:
    - x: Input feature.
    - y: Target variable.
    - T: Coefficients of the polynomial fit.
    - S: Coefficients of the seasonal curve fit.

    Returns:
    - y_fitted: Fitted values.
    - fitting_error: Error in fitting.
    - fitted_function: Function representing the complete fit.
    """

    def fitted_function(x):
        return polynomial_curve_function(x, T[0], T[1], T[2], T[3]) + seasonal_curve_function(x, S[0], S[1], S[2], S[3])

    y_fitted = np.squeeze(fitted_function(x))
    fitting_error = y - y_fitted
    return y_fitted, fitting_error, fitted_function


def check_fitting_quality_and_print_metrics(y, y_fitted):
    """
    Calculate and print metrics to assess fitting quality.

    Args:
    - y: True values.
    - y_fitted: Fitted values.

    Returns:
    None
    """
    mse = mean_squared_error(y, y_fitted)
    rmse = np.sqrt(mse)
    r_squared = r2_score(y, y_fitted)
    print(f'Mean Squared Error: {mse}')
    print(f'Root Mean Squared Error: {rmse}')
    print(f'R-squared: {r_squared}')


def print_error_distribution_and_return_stats(fitting_error):
    """
    Print error distribution statistics.

    Args:
    - fitting_error: Error in fitting.

    Returns:
    - error_mean: Mean of the fitting error.
    - error_std: Standard deviation of the fitting error.
    """
    error_mean, error_std = norm.fit(fitting_error)
    print(f"Mean of the error: {error_mean}")
    print(f"Standard deviation of the error: {error_std}")
    return error_mean, error_std


def reevaluate_model(data):
    """
    Reevaluate the model using the provided data.

    Args:
    - data: List of tuples containing timestamp and corresponding values.

    Returns:
    - fitted_function_: Fitted function representing the complete fit.
    - e_std: Standard deviation of the fitting error.
    """
    df = format_data(data)
    cycle, trend, optimal_lambda = apply_hp_filter_with_optimal_lambda(time_series=df['value'],
                                                                       lambda_range=[1, 10, 50, 100, 200, 500, 0.5])
    result = seasonal_decomposition(time_series=trend, period_range=[10, 20, 50, 100, 200, 500, 1000, 2000])
    df_trend = df.copy()
    df_trend["value"] = result.trend
    df_trend = df_trend.dropna()
    poly_coef_ = polynomial_fit_and_plot(x=df_trend["timestamp_sec"].values.reshape(-1, 1),
                                         y=df_trend["value"].values.reshape(-1, 1), degree=1)
    period_coef_ = seasonal_curve_fit_and_plot(f=seasonal_curve_function, x=df['timestamp_sec'], y=result.seasonal)
    y_fitted_, fitting_error_, fitted_function_ = complete_fit_and_plot(df["timestamp_sec"], df["value"], poly_coef_,
                                                                        period_coef_)
    check_fitting_quality_and_print_metrics(df["value"], y_fitted_)
    e_mean, e_std = print_error_distribution_and_return_stats(fitting_error_)
    return poly_coef_.tolist(), period_coef_.tolist(), e_std
