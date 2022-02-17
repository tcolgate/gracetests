import os

from setuptools import find_packages, setup

DIR = os.path.dirname(os.path.abspath(__file__))

setup(
    name='graceful',
    version='1.2.0',
    description='test of graceful shutdown',
    url='https://github.com/qubitdigital/graceful/python',
    author='Infra',
    author_email='infra@qubit.com',
    license='All rights reserved.',
    packages=find_packages(),
    install_requires=[
        'sanic==20.12.6',
        'ujson==1.35',
        'python-dotenv==0.8.2',
        'cchardet==2.1.1',
    ],
    zip_safe=False,
    entry_points={
        'console_scripts': [
            'graceful=graceful.server:main',
        ]
    }
)
